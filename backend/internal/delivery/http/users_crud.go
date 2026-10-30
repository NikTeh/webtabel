package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool" // ССЫЛКА
)

// OwnerUserCrudResponse описывает структуру данных пользователя для CRUD-контура Владельца
type OwnerUserCrudResponse struct {
	ID             int    `json:"id"`
	ExternalID     *int   `json:"external_id"`
	Phone          string `json:"phone"`
	Password       string `json:"password"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	ManufacturerID *int   `json:"manufacturer_id"` // Добавлено поле для связывания с Производителем
}

// ManageUsersHandler реализует точечный CRUD-контур ручного управления пользователями силами Владельца
func ManageUsersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			// Вывод списка абсолютно всех пользователей (кроме главного суперадмина) для таблицы
			query := `
				SELECT id, external_id, phone, password, full_name, role, manufacturer_id 
				FROM users 
				WHERE role != 'owner'
				ORDER BY role, id DESC`

			rows, err := db.Query(ctx, query)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка получения справочника пользователей из БД"}`))
				return
			}
			defer rows.Close()

			usersList := []OwnerUserCrudResponse{}
			for rows.Next() {
				var u OwnerUserCrudResponse
				if err := rows.Scan(&u.ID, &u.ExternalID, &u.Phone, &u.Password, &u.FullName, &u.Role, &u.ManufacturerID); err != nil {
					continue
				}
				usersList = append(usersList, u)
			}
			json.NewEncoder(w).Encode(usersList)
			return

		case http.MethodPost:
			// 2. РУЧНОЕ ТОЧЕЧНОЕ ДОБАВЛЕНИЕ ОДНОГО ПОЛЬЗОВАТЕЛЯ ВЛАДЕЛЬЦЕМ
			var req OwnerUserCrudResponse
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
				return
			}

			req.Phone = strings.TrimSpace(req.Phone)
			if req.Phone == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Заполните обязательные поля: Телефон, Пароль, ФИО и Роль"}`))
				return
			}

			// Проверяем: если роль не Производитель, принудительно обнуляем manufacturer_id
			if req.Role != "manufacturer" {
				req.ManufacturerID = nil
			}

			insertQuery := `
				INSERT INTO users (external_id, phone, password, full_name, role, manufacturer_id)
				VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

			var newUserID int
			err := db.QueryRow(ctx, insertQuery, req.ExternalID, req.Phone, req.Password, req.FullName, req.Role, req.ManufacturerID).Scan(&newUserID)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					w.WriteHeader(http.StatusConflict)
					w.Write([]byte(`{"error": "Пользователь с таким номером телефона или ID 1С уже зарегистрирован"}`))
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка добавления пользователя в СУБД"}`))
				return
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "success", "message": "Пользователь успешно добавлен", "id": ` + strconv.Itoa(newUserID) + `}`))
			return

		case http.MethodPut:
			// 3. РУЧНОЕ ТОЧЕЧНОЕ ИЗМЕНЕНИЕ ДАННЫХ ПОЛЬЗОВАТЕЛЯ ВЛАДЕЛЬЦЕМ
			var req OwnerUserCrudResponse
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
				return
			}

			if req.ID <= 0 || req.Phone == "" || req.FullName == "" || req.Role == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Не указан ID или пустые обязательные поля"}`))
				return
			}

			if req.Role != "manufacturer" {
				req.ManufacturerID = nil
			}

			updateQuery := `
				UPDATE users 
				SET phone = $1, password = $2, full_name = $3, role = $4, external_id = $5, manufacturer_id = $6, updated_at = NOW()
				WHERE id = $7`

			_, err := db.Exec(ctx, updateQuery, req.Phone, req.Password, req.FullName, req.Role, req.ExternalID, req.ManufacturerID, req.ID)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					w.WriteHeader(http.StatusConflict)
					w.Write([]byte(`{"error": "Этот ID из 1С или телефон уже заняты другим пользователем"}`))
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка обновления данных пользователя"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Данные пользователя успешно изменены"}`))
			return

		case http.MethodDelete:
			// 4. РУЧНОЕ УДАЛЕНИЕ ПОЛЬЗОВАТЕЛЯ ВЛАДЕЛЬЦЕМ
			idStr := r.URL.Query().Get("id")
			id, err := strconv.Atoi(idStr)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Некорректный или отсутствующий параметр ID"}`))
				return
			}

			deleteQuery := `DELETE FROM users WHERE id = $1 AND role != 'owner'`
			_, err = db.Exec(ctx, deleteQuery, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка удаления из БД (возможно, за пользователем закреплены активные связи)"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Пользователь успешно удален из системы"}`))
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
		}
	}
}
