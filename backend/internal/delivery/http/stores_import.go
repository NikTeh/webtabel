package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2" // ССЫЛКА
)

// ImportStoresHandler обрабатывает импорт XLSX файла по принципу "Добавить новые / Обновить старые"
func ImportStoresHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// if r.Method != http.MethodPost {
		// 	w.WriteHeader(http.StatusMethodNotAllowed)
		// 	w.Write([]byte(`{"error": "Метод не поддерживается"}`))
		// 	return
		// }
		// Разрешаем бэкенду принимать и POST (для импорта файлов), и GET (для показа таблицы на сайте)
		// Разрешаем бэкенду принимать и POST (для импорта файлов), и GET (для показа таблицы на сайте)
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается сервером"}`))
			return
		}

		// Если Владелец просто открыл вкладку, отдаем ему реальный список точек из PostgreSQL
		if r.Method == http.MethodGet {
			query := `
				SELECT s.id_1c, s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
					COALESCE(sv.full_name, 'Не назначен') AS sv_name,
					COALESCE(sv.phone, '') AS sv_phone
				FROM stores s
				LEFT JOIN users sv ON s.supervisor_id = sv.id AND sv.role = 'supervisor'
				ORDER BY s.id_1c DESC`

			rows, err := db.Query(ctx, query)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка чтения списка магазинов из PostgreSQL"}`))
				return
			}
			defer rows.Close()

			list := []map[string]interface{}{}
			for rows.Next() {
				var id1c, region, city, network, sign, format, address, svName, svPhone string
				if err := rows.Scan(&id1c, &region, &city, &network, &sign, &format, &address, &svName, &svPhone); err == nil {
					list = append(list, map[string]interface{}{
						"id_1c":          id1c,
						"region":         region,
						"city":           city,
						"retail_network": network,
						"store_sign":     sign,
						"store_format":   format,
						"address":        address,
						"sv_name":        svName,
						"sv_phone":       svPhone,
					})
				}
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		// Техническая защита: лимит 100 МБ на файл по условиям проекта
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не удалось прочитать файл в поле 'file'"}`))
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
			w.Write([]byte(`{"error": "В таблице отсутствуют строки с данными"}`))
			return
		}

		// Открываем транзакцию для безопасной пакетной записи
		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось открыть транзакцию базы данных"}`))
			return
		}
		defer tx.Rollback(ctx)

		// СТРОКА ОЧИСТКИ DELETE УДАЛЕНА НАВСЕГДА ДЛЯ СОХРАНЕНИЯ ИСТОРИИ

		insertedCount := 0
		updatedCount := 0

		// Читаем строки файла Excel (пропуская заголовок на индексе 0)
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 9 {
				continue // Пропускаем неполные строки файла
			}

			id1c := strings.TrimSpace(row[0])
			region := strings.TrimSpace(row[1])
			city := strings.TrimSpace(row[2])
			retailNetwork := strings.TrimSpace(row[3])
			storeSign := strings.TrimSpace(row[4])
			storeFormat := strings.TrimSpace(row[5])
			address := strings.TrimSpace(row[6])
			svPhone := strings.TrimSpace(row[8]) // Уникальный телефон супервайзера для связывания

			if id1c == "" || region == "" || city == "" || retailNetwork == "" || storeSign == "" || storeFormat == "" || address == "" || svPhone == "" {
				continue // Игнорируем строки с пустыми важными полями
			}

			// Читаем опциональные геокоординаты из колонок 10 и 11 (индексы 9 и 10)
			var lat, lon *float64
			if len(row) > 9 {
				if lVal, err := strconv.ParseFloat(strings.TrimSpace(row[9]), 64); err == nil {
					lat = &lVal
				}
			}
			if len(row) > 10 {
				if lVal, err := strconv.ParseFloat(strings.TrimSpace(row[10]), 64); err == nil {
					lon = &lVal
				}
			}

			// Проверяем, загружен ли уже такой супервайзер в систему
			var svInternalID int
			err = tx.QueryRow(ctx, "SELECT id FROM users WHERE phone = $1 AND role = 'supervisor' LIMIT 1", svPhone).Scan(&svInternalID)
			if err != nil {
				continue
			}

			// Пишем "Умный" SQL-запрос с поддержкой обновления географических координат точек
			upsertQuery := `
				INSERT INTO stores (id_1c, region, city, retail_network, store_sign, store_format, address, supervisor_id, merchandiser_id, latitude, longitude)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9, $10)
				ON CONFLICT (id_1c) 
				DO UPDATE SET 
					region = EXCLUDED.region,
					city = EXCLUDED.city,
					retail_network = EXCLUDED.retail_network,
					store_sign = EXCLUDED.store_sign,
					store_format = EXCLUDED.store_format,
					address = EXCLUDED.address,
					supervisor_id = EXCLUDED.supervisor_id,
					latitude = EXCLUDED.latitude,
					longitude = EXCLUDED.longitude,
					updated_at = NOW()
				RETURNING (xmax = 0)`

			var isInserted bool
			err = tx.QueryRow(ctx, upsertQuery, id1c, region, city, retailNetwork, storeSign, storeFormat, address, svInternalID, lat, lon).Scan(&isInserted)
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
			w.Write([]byte(`{"error": "Ошибка фиксации изменений"}`))
			return
		}

		// Возвращаем честный аналитический статус владельцу в ЛК
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "inserted": ` + strconv.Itoa(insertedCount) + `, "updated": ` + strconv.Itoa(updatedCount) + `}`))
	}
}
