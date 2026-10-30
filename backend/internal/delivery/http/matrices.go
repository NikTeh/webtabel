package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool" // ССЫЛКА
)

// MatrixItemRecord описывает одну ячейку матрицы (установленный чекбокс в БД)
type MatrixItemRecord struct {
	SkuID         string `json:"sku_id"`
	RetailNetwork string `json:"retail_network"`
	StoreFormat   string `json:"store_format"`
}

// SaveMatrixRequest описывает структуру входящего JSON-пакета для сохранения матрицы владельцем
type SaveMatrixRequest struct {
	ManufacturerID int                `json:"manufacturer_id"`
	Items          []MatrixItemRecord `json:"items"`
}

// GetMatrixHandler возвращает текущую сохраненную матрицу производителя для отрисовки сетки в ЛК
func GetMatrixHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
			return
		}

		mIDStr := r.URL.Query().Get("manufacturer_id")
		mID, err := strconv.Atoi(mIDStr)
		if err != nil || mID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указан или некорректен параметр manufacturer_id"}`))
			return
		}

		// Выбираем из плоской таблицы store_matrices все сохраненные чекбоксы для этого производителя
		query := `
			SELECT sku_id, retail_network, store_format 
			FROM store_matrices 
			WHERE manufacturer_id = $1`

		rows, err := db.Query(ctx, query, mID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка чтения матрицы"}`))
			return
		}
		defer rows.Close()

		items := []MatrixItemRecord{}
		for rows.Next() {
			var item MatrixItemRecord
			if err := rows.Scan(&item.SkuID, &item.RetailNetwork, &item.StoreFormat); err == nil {
				items = append(items, item)
			}
		}

		// Возвращаем плоский массив включенных галочек. Фронтенд сам сопоставит их с SKU-листом.
		json.NewEncoder(w).Encode(items)
	}
}

// SaveMatrixHandler полностью перезаписывает матрицу производителя в рамках единой транзакции
func SaveMatrixHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
			return
		}

		var req SaveMatrixRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON пакета"}`))
			return
		}

		if req.ManufacturerID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указан корректный идентификатор manufacturer_id"}`))
			return
		}

		// Открываем транзакцию для безопасной перезаписи без потери данных при сбоях сети
		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось открыть транзакцию базы данных"}`))
			return
		}
		defer tx.Rollback(ctx)

		// 1. Очищаем абсолютно все старые строки матрицы для этого производителя
		_, err = tx.Exec(ctx, "DELETE FROM store_matrices WHERE manufacturer_id = $1", req.ManufacturerID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка очистки предыдущей матрицы"}`))
			return
		}

		// 2. Записываем только те чекбоксы, которые сейчас прислал фронтенд (где стоят галочки)
		insertedCount := 0
		for _, item := range req.Items {
			if item.SkuID == "" || item.RetailNetwork == "" || item.StoreFormat == "" {
				continue
			}

			insertQuery := `
				INSERT INTO store_matrices (manufacturer_id, sku_id, retail_network, store_format)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT DO NOTHING`

			_, err = tx.Exec(ctx, insertQuery, req.ManufacturerID, item.SkuID, item.RetailNetwork, item.StoreFormat)
			if err != nil {
				// Если SKU был изменен или удален — пропускаем строку ради стабильности транзакции
				continue
			}
			insertedCount++
		}

		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка сохранения транзакции матрицы"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "Матрица успешно сохранена", "saved_checkboxes_count": ` + strconv.Itoa(insertedCount) + `}`))
	}
}
