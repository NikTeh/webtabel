package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2" // ССЫЛКА
)

// OwnerUserResponse описывает структуру данных пользователя для интерфейса Владельца
type OwnerUserResponse struct {
	ID             int    `json:"id"`
	ExternalID     *int   `json:"external_id"`
	Phone          string `json:"phone"`
	Password       string `json:"password"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	ManufacturerID *int   `json:"manufacturer_id"`
}

// ImportUsersHandler обрабатывает загрузку XLSX файла пользователей по принципу Upsert (Добавить/Обновить)
func ImportUsersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// МЕТОД GET: Вывод справочника всех пользователей для Владельца
		if r.Method == http.MethodGet {
			query := `
				SELECT id, external_id, phone, password, full_name, role, manufacturer_id 
				FROM users 
				WHERE role != 'owner'
				ORDER BY role, id DESC`

			rows, err := db.Query(ctx, query)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка получения справочника пользователей"}`))
				return
			}
			defer rows.Close()

			usersList := []OwnerUserResponse{}
			for rows.Next() {
				var u OwnerUserResponse
				if err := rows.Scan(&u.ID, &u.ExternalID, &u.Phone, &u.Password, &u.FullName, &u.Role, &u.ManufacturerID); err != nil {
					continue
				}
				usersList = append(usersList, u)
			}
			json.NewEncoder(w).Encode(usersList)
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
			return
		}

		// Техническая защита: Ограничение размера файла в 100 МБ по ТЗ
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не удалось прочитать файл"}`))
			return
		}
		defer file.Close()

		xlsx, err := excelize.OpenReader(file)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат XLSX файла"}`))
			return
		}
		defer xlsx.Close()

		sheetName := xlsx.GetSheetName(0)
		rows, err := xlsx.GetRows(sheetName)
		if err != nil || len(rows) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Выбранный лист Excel пуст"}`))
			return
		}

		if len(rows) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "В таблице отсутствуют данные для импорта"}`))
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось открыть транзакцию БД"}`))
			return
		}
		defer tx.Rollback(ctx)

		// СТРОКА ОЧИСТКИ DELETE УДАЛЕНА. СТАРЫЕ ПОЛЬЗОВАТЕЛИ БОЛЬШЕ НЕ СТИРАЮТСЯ.

		insertedCount := 0
		updatedCount := 0

		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 5 {
				continue
			}

			extIDStr := strings.TrimSpace(row[0])
			phone := strings.TrimSpace(row[1])
			password := strings.TrimSpace(row[2])
			fullName := strings.TrimSpace(row[3])
			roleStr := strings.TrimSpace(row[4])

			if phone == "" || password == "" || fullName == "" || roleStr == "" {
				continue
			}

			var extID *int
			if val, err := strconv.Atoi(extIDStr); err == nil {
				extID = &val
			}

			var role string
			switch roleStr {
			case "Руководитель", "owner":
				role = "owner"
			case "Супервайзер", "supervisor", "sv":
				role = "supervisor"
			case "Мерчандайзер", "merchandiser", "me":
				role = "merchandiser"
			case "Производитель", "manufacturer":
				role = "manufacturer"
			default:
				continue // Неизвестную роль пропускаем, чтобы не вызвать ошибку ограничения CHECK
			}

			// Пропускаем попытку повторно импортировать роль owner, чтобы избежать конфликтов сессий
			if role == "owner" {
				continue
			}

			var manufacturerID interface{} = nil
			if len(row) >= 6 && role == "manufacturer" {
				mIDStr := strings.TrimSpace(row[5])
				if mIDStr != "" {
					if parsedID, err := strconv.Atoi(mIDStr); err == nil {
						manufacturerID = parsedID
					}
				}
			}

			// Умный запрос: если телефон (логин) совпал, обновляем ФИО, пароль и роль, не дублируя сотрудника
			upsertQuery := `
				INSERT INTO users (external_id, phone, password, full_name, role, manufacturer_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (phone) 
				DO UPDATE SET 
					external_id = EXCLUDED.external_id,
					password = EXCLUDED.password,
					full_name = EXCLUDED.full_name,
					role = EXCLUDED.role,
					manufacturer_id = EXCLUDED.manufacturer_id,
					updated_at = NOW()
				RETURNING (xmax = 0)`

			var isInserted bool
			err = tx.QueryRow(ctx, upsertQuery, extID, phone, password, fullName, role, manufacturerID).Scan(&isInserted)
			if err != nil {
				continue
			}

			if isInserted {
				insertedCount++
			} else {
				updatedCount++
			}
		}

		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось зафиксировать транзакцию пользователей"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "inserted": ` + strconv.Itoa(insertedCount) + `, "updated": ` + strconv.Itoa(updatedCount) + `}`))
	}
}
