package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SvMeRequest описывает структуру данных мерчандайзера от интерфейса СВ
type SvMeRequest struct {
	ID         int    `json:"id"`
	ExternalID *int   `json:"external_id"` // ИСПРАВЛЕНО: Добавлено поле приема кода из 1С
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
}

// ManageSvMerchandisersHandler реализует изолированный CRUD-контур управления МЕ силами Супервайзера
func ManageSvMerchandisersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// Извлекаем текущего СВ из контекста
		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
			return
		}

		switch r.Method {
		case http.MethodGet:
			// 1. ПОЛУЧЕНИЕ СПИСКА ТОЛЬКО СВОИХ МЕ (Пункт 21.1 ТЗ)
			// ИСПРАВЛЕНО: Добавлено чтение u.external_id из таблицы users
			query := `
				SELECT u.id, u.external_id, u.full_name, u.phone, u.password 
				FROM users u
				INNER JOIN supervisor_merchandisers sm ON u.id = sm.merchandiser_id
				WHERE sm.supervisor_id = $1 AND u.role = 'merchandiser'`

			rows, err := db.Query(ctx, query, user.ID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка получения списка мерчандайзеров"}`))
				return
			}
			defer rows.Close()

			mes := []SvMeRequest{}
			for rows.Next() {
				var me SvMeRequest
				// ИСПРАВЛЕНО: Считываем external_id в структуру
				if err := rows.Scan(&me.ID, &me.ExternalID, &me.FullName, &me.Phone, &me.Password); err != nil {
					continue
				}
				mes = append(mes, me)
			}
			json.NewEncoder(w).Encode(mes)

		case http.MethodPost:
			// 2. ДОБАВЛЕНИЕ НОВОГО МЕ С ПРИВЯЗКОЙ К СВ
			var req SvMeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
				return
			}

			req.Phone = strings.TrimSpace(req.Phone)
			// ИСПРАВЛЕНО: Добавлена проверка на обязательное заполнение кода 1С
			if req.Phone == "" || req.Password == "" || req.FullName == "" || *req.ExternalID <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Все обязательные ячейки должны быть заполнены (включая ID из 1С)"}`))
				return
			}

			// Открываем транзакцию, чтобы добавление МЕ и привязка к СВ прошли атомарно
			tx, err := db.Begin(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка открытия транзакции"}`))
				return
			}
			defer tx.Rollback(ctx)

			// Создаем запись мерчандайзера в таблице users
			var newMeID int
			// ИСПРАВЛЕНО: Заменили зашитый NULL на динамический параметр \$1 (req.ExternalID)
			insertUserQuery := `
				INSERT INTO users (external_id, phone, password, full_name, role, manufacturer_id)
				VALUES ($1, $2, $3, $4, 'merchandiser', NULL) RETURNING id`

			err = tx.QueryRow(ctx, insertUserQuery, req.ExternalID, req.Phone, req.Password, req.FullName).Scan(&newMeID)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					w.WriteHeader(http.StatusConflict)
					w.Write([]byte(`{"error": "Мерчандайзер с таким номером телефона или ID 1С уже зарегистрирован"}`))
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка записи мерчандайзера в БД"}`))
				return
			}

			// Жестко связываем нового МЕ с текущим Супервайзером
			insertLinkQuery := `
				INSERT INTO supervisor_merchandisers (supervisor_id, merchandiser_id)
				VALUES ($1, $2)`

			_, err = tx.Exec(ctx, insertLinkQuery, user.ID, newMeID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка создания связи СВ-МЕ в БД"}`))
				return
			}

			if err := tx.Commit(ctx); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Не удалось зафиксировать транзакцию"}`))
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "success", "message": "Мерчандайзер успешно добавлен", "id": ` + strconv.Itoa(newMeID) + `}`))

		case http.MethodPut:
			// 3. ИЗМЕНЕНИЕ ДАННЫХ МЕ (ФИО, ТЕЛЕФОН, ПАРОЛЬ)
			var req SvMeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
				return
			}

			if req.ID <= 0 || req.Phone == "" || req.FullName == "" || *req.ExternalID <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Не указан ID или пустые обязательные поля"}`))
				return
			}

			// Безопасность: Проверяем, действительно ли этот МЕ принадлежит текущему СВ
			var belongs bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM supervisor_merchandisers WHERE supervisor_id = $1 AND merchandiser_id = $2)`
			err := db.QueryRow(ctx, checkQuery, user.ID, req.ID).Scan(&belongs)
			if err != nil || !belongs {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Доступ запрещен. Этот мерчандайзер не входит в вашу команду"}`))
				return
			}

			// ИСПРАВЛЕНО: Добавлено обновление external_id при редактировании сотрудника
			updateQuery := `
				UPDATE users 
				SET phone = $1, password = $2, full_name = $3, external_id = $4, updated_at = NOW()
				WHERE id = $5 AND role = 'merchandiser'`

			_, err = db.Exec(ctx, updateQuery, req.Phone, req.Password, req.FullName, req.ExternalID, req.ID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка обновления данных мерчандайзера в БД"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Данные мерчандайзера успешно изменены"}`))

		case http.MethodDelete:
			// 4. УДАЛЕНИЕ МЕ СУПЕРВАЙЗЕРОМ
			idStr := r.URL.Query().Get("id")
			id, err := strconv.Atoi(idStr)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный или отсутствующий параметр ID"}`))
				return
			}

			// Безопасность: Проверяем принадлежность МЕ к текущему СВ перед удалением
			var belongs bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM supervisor_merchandisers WHERE supervisor_id = $1 AND merchandiser_id = $2)`
			err = db.QueryRow(ctx, checkQuery, user.ID, id).Scan(&belongs)
			if err != nil || !belongs {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Доступ запрещен. Невозможно удалить чужого мерчандайзера"}`))
				return
			}

			// Удаление записи
			deleteQuery := `DELETE FROM users WHERE id = $1 AND role = 'merchandiser'`
			_, err = db.Exec(ctx, deleteQuery, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка удаления мерчандайзера из БД (возможно, за ним закреплены активные визиты)"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Мерчандайзер успешно удален из вашей команды"}`))

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
		}
	}
}
