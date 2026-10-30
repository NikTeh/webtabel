package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2" // ССЫЛКА
)

// ManufacturerRequest описывает структуру создания карточки производителя
type ManufacturerRequest struct {
	Name            string `json:"name"`
	WorkDays        []int  `json:"work_days"`
	DurationMinutes int    `json:"duration_minutes"`
}

// ManageManufacturersHandler отвечает за ручное создание и получение списка производителей
func ManageManufacturersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method == http.MethodGet {
			query := `SELECT id, name, work_days, duration_minutes FROM manufacturers ORDER BY name ASC`
			rows, err := db.Query(ctx, query)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка чтения списка производителей из БД"}`))
				return
			}
			defer rows.Close()

			list := []map[string]interface{}{}
			for rows.Next() {
				var id int
				var name string
				var workDays []int
				var duration int
				if err := rows.Scan(&id, &name, &workDays, &duration); err == nil {
					list = append(list, map[string]interface{}{
						"id":               id,
						"name":             name,
						"work_days":        workDays,
						"duration_minutes": duration,
					})
				}
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
			return
		}

		var req ManufacturerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" || len(req.WorkDays) == 0 || req.DurationMinutes <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Все поля карточки обязательны"}`))
			return
		}

		query := `INSERT INTO manufacturers (name, work_days, duration_minutes) VALUES ($1, $2, $3) RETURNING id`
		var newID int
		err := db.QueryRow(ctx, query, req.Name, req.WorkDays, req.DurationMinutes).Scan(&newID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка записи производителя в БД"}`))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "success", "id": ` + strconv.Itoa(newID) + `}`))
	}
}

// ImportSkusHandler обрабатывает XLSX импорт и GET-выгрузку SKU Производителя для ЛК Владельца
func ImportSkusHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		mIDStr := r.FormValue("manufacturer_id")
		if mIDStr == "" {
			mIDStr = r.URL.Query().Get("manufacturer_id")
		}
		mID, err := strconv.Atoi(mIDStr)
		if err != nil || mID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указан или некорректен параметр manufacturer_id"}`))
			return
		}

		// МЕТОД GET: Возвращает реальный SKU-лист из PostgreSQL для вкладки "Инфо" и матриц
		if r.Method == http.MethodGet {
			query := `SELECT id, name, brand, category FROM skus WHERE manufacturer_id = $1 ORDER BY category, name`
			rows, err := db.Query(ctx, query, mID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка получения списка SKU из БД"}`))
				return
			}
			defer rows.Close()

			list := []map[string]interface{}{}
			for rows.Next() {
				var id, name, brand, category string
				if err := rows.Scan(&id, &name, &brand, &category); err == nil {
					list = append(list, map[string]interface{}{
						"id":       id,
						"name":     name,
						"brand":    brand,
						"category": category,
					})
				}
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		// Логика POST импорта XLSX (из Сообщения №28) остается неизменной
		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не удалось получить файл"}`))
			return
		}
		defer file.Close()

		xlsx, err := excelize.OpenReader(file)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат XLSX"}`))
			return
		}
		defer xlsx.Close()

		rows, _ := xlsx.GetRows(xlsx.GetSheetName(0))
		tx, _ := db.Begin(ctx)
		defer tx.Rollback(ctx)

		insertedCount, updatedCount := 0, 0
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 5 {
				continue
			}
			skuID := strings.TrimSpace(row[0])
			skuName := strings.TrimSpace(row[2])
			brand := strings.TrimSpace(row[3])
			category := strings.TrimSpace(row[4])

			upsertQuery := `
				INSERT INTO skus (id, manufacturer_id, name, brand, category)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, brand = EXCLUDED.brand, category = EXCLUDED.category
				RETURNING (xmax = 0)`

			var isInserted bool
			if err := tx.QueryRow(ctx, upsertQuery, skuID, mID, skuName, brand, category).Scan(&isInserted); err == nil {
				if isInserted {
					insertedCount++
				} else {
					updatedCount++
				}
			}
		}
		tx.Commit(ctx)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "inserted": ` + strconv.Itoa(insertedCount) + `, "updated": ` + strconv.Itoa(updatedCount) + `}`))
	}
}

// ImportManufacturerStoresHandler обрабатывает XLSX импорт и GET-выгрузку адресной программы Производителя
func ImportManufacturerStoresHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		mIDStr := r.URL.Query().Get("manufacturer_id")
		if mIDStr == "" {
			mIDStr = r.FormValue("manufacturer_id")
		}
		mID, err := strconv.Atoi(mIDStr)
		if err != nil || mID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указан или некорректен параметр manufacturer_id"}`))
			return
		}

		// МЕТОД GET: Возвращает реальный список магазинов, привязанных к производителю
		if r.Method == http.MethodGet {
			query := `
				SELECT s.id_1c, s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
				       COALESCE(sv.full_name, 'Не назначен') as sv_name, COALESCE(sv.phone, '') as sv_phone
				FROM stores s
				INNER JOIN manufacturer_stores ms ON s.id_1c = ms.store_id_1c
				LEFT JOIN users sv ON s.supervisor_id = sv.id
				WHERE ms.manufacturer_id = $1
				ORDER BY s.retail_network, s.id_1c`

			rows, err := db.Query(ctx, query, mID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка чтения точек производителя из БД"}`))
				return
			}
			defer rows.Close()

			list := []map[string]interface{}{}
			for rows.Next() {
				var id, region, city, network, sign, format, address, svName, svPhone string
				if err := rows.Scan(&id, &region, &city, &network, &sign, &format, &address, &svName, &svPhone); err == nil {
					list = append(list, map[string]interface{}{
						"id_1c":          id,
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

		// Логика POST импорта (из Сообщения №28) остается неизменной
		file, _, _ := r.FormFile("file")
		defer file.Close()
		xlsx, _ := excelize.OpenReader(file)
		defer xlsx.Close()

		rows, _ := xlsx.GetRows(xlsx.GetSheetName(0))
		tx, _ := db.Begin(ctx)
		defer tx.Rollback(ctx)

		insertedCount := 0
		for i := 1; i < len(rows); i++ {
			if len(rows[i]) == 0 {
				continue
			}
			id1c := strings.TrimSpace(rows[i][1]) // Индекс 1 в соответствии с ТЗ импорта производителя
			var exists bool
			_ = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM stores WHERE id_1c = $1)", id1c).Scan(&exists)
			if exists {
				cmdTag, err := tx.Exec(ctx, "INSERT INTO manufacturer_stores (manufacturer_id, store_id_1c) VALUES ($1, $2) ON CONFLICT DO NOTHING", mID, id1c)
				if err == nil && cmdTag.RowsAffected() > 0 {
					insertedCount++
				}
			}
		}
		tx.Commit(ctx)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "inserted": ` + strconv.Itoa(insertedCount) + `}`))
	}
}
