package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartVisitRequest описывает входящие данные о начале визита с Android-устройства
type StartVisitRequest struct {
	ID             string    `json:"id"` // UUID, сгенерированный на Android
	StoreId1c      string    `json:"store_id_1c"`
	StartedAt      time.Time `json:"started_at"`
	IsScheduled    bool      `json:"is_scheduled"`
	IsOnline       bool      `json:"is_online"`
	StartLatitude  *float64  `json:"start_latitude"`
	StartLongitude *float64  `json:"start_longitude"`
}

// FinishVisitRequest описывает входящие данные о завершении визита
type FinishVisitRequest struct {
	ID         string `json:"id"` // UUID визита
	FinishedAt string `json:"finished_at"`
}

// StartVisitHandler фиксирует открытие визита мерчандайзером в точке
func StartVisitHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// Извлекаем текущего МЕ из сквозного контекста авторизации
		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
			return
		}

		var req StartVisitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		if req.ID == "" || req.StoreId1c == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указаны ID визита или ID точки"}`))
			return
		}

		// query := `
		// 	INSERT INTO visits (id, store_id_1c, merchandiser_id, started_at, finished_at, is_scheduled, is_online, start_latitude, start_longitude, status, finish_type)
		// 	VALUES ($1, $2, $3, $4, NULL, $5, $6, $7, $8, 'active', NULL)
		// 	ON CONFLICT (id) DO NOTHING`
		query := `
	INSERT INTO visits (id, store_id_1c, merchandiser_id, started_at, is_scheduled, is_online, start_latitude, start_longitude, status, finish_type)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', NULL)
	ON CONFLICT (id) DO NOTHING`

		_, err := db.Exec(ctx, query, req.ID, req.StoreId1c, user.ID, req.StartedAt, req.IsScheduled, req.IsOnline, req.StartLatitude, req.StartLongitude)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка фиксации старта визита в БД"}`))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "success", "message": "Старт визита успешно зафиксирован"}`))
	}
}

// FinishVisitHandler переводит визит в статус завершенного
func FinishVisitHandler(db *pgxpool.Pool) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		var req FinishVisitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		fmt.Println(req.FinishedAt)

		query := `
			UPDATE visits
			SET ended_at = $1::TIMESTAMP, status = 'finished', finish_type = 'normal'
			WHERE id = $2 AND status = 'active'`

		cmdTag, err := db.Exec(ctx, query, req.FinishedAt, req.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка закрытия визита в БД"}`))
			fmt.Println("err")
			fmt.Println(err)
			return
		}

		if cmdTag.RowsAffected() == 0 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Активный визит с таким ID не найден"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "Визит успешно завершен"}`))
	}
}
