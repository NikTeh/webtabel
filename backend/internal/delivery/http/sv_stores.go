package http

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AssignMeRequest описывает структуру запроса на закрепление МЕ за точкой
type AssignMeRequest struct {
	StoreId1c      string `json:"store_id_1c"`
	MerchandiserID *int   `json:"merchandiser_id"` // Может быть NULL, если МЕ убирают с точки
}

// ManufacturerSchedule описывает график визитов конкретного бренда на торговую точку
type ManufacturerSchedule struct {
	Name            string `json:"name"`
	WorkDays        string `json:"work_days"`
	DurationMinutes int    `json:"duration_minutes"`
}

// SvStoreRow описывает расширенную структуру строки торговой точки для ЛК Супервайзера
type SvStoreRow struct {
	Id1c           string                 `json:"id_1c"`
	Region         string                 `json:"region"`
	City           string                 `json:"city"`
	RetailNetwork  string                 `json:"retail_network"`
	StoreSign      string                 `json:"store_sign"`
	StoreFormat    string                 `json:"store_format"`
	Address        string                 `json:"address"`
	MerchandiserID *int                   `json:"merchandiser_id"`
	MeFullName     string                 `json:"me_full_name"`
	MePhone        string                 `json:"me_phone"`  // НОВЫЙ ФУНКЦИОНАЛ: Телефон МЕ для связи
	Schedules      []ManufacturerSchedule `json:"schedules"` // НОВЫЙ ФУНКЦИОНАЛ: Сжатый массив графиков производителей
}

// GetSvStoresHandler возвращает расширенный список точек с производителями и МЕ для Супервайзера
func GetSvStoresHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// Достаем авторизованного СВ из сквозного контекста
		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
			return
		}

		// ИСПРАВЛЕНО: Безопасный высокопроизводительный SQL-запрос с LEFT JOIN и JSON-агрегацией
		// Он собирает расписания всех производителей в одну компактную строку СУБД, исключая дублирование точек
		query := `
			SELECT 
				s.id_1c, s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address, s.merchandiser_id,
				COALESCE(u.full_name, 'Вакансия') AS me_full_name,
				COALESCE(u.phone, '') AS me_phone,
				COALESCE(
					json_agg(
						json_build_object(
							'name', m.name,
							'work_days', m.work_days::TEXT,
							'duration_minutes', m.duration_minutes
						)
					) FILTER (WHERE m.id IS NOT NULL), '[]'::json
				) AS schedules
			FROM stores s
			LEFT JOIN users u ON s.merchandiser_id = u.id AND u.role = 'merchandiser'
			LEFT JOIN manufacturer_stores ms ON s.id_1c = ms.store_id_1c
			LEFT JOIN manufacturers m ON ms.manufacturer_id = m.id
			WHERE s.supervisor_id = $1
			GROUP BY s.id_1c, s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address, s.merchandiser_id, u.full_name, u.phone
			ORDER BY s.retail_network, s.id_1c DESC`

		rows, err := db.Query(ctx, query, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка чтения точек и графиков производителей из базы данных"}`))
			return
		}
		defer rows.Close()

		stores := []SvStoreRow{}
		for rows.Next() {
			var s SvStoreRow
			var jsonSchedules []byte // Прослойка байт для безопасного чтения агрегированного JSON СУБД

			// ИСПРАВЛЕНО: Безопасное потоковое считывание данных, включая телефон МЕ и JSON-пакет брендов
			if err := rows.Scan(
				&s.Id1c, &s.Region, &s.City, &s.RetailNetwork, &s.StoreSign, &s.StoreFormat, &s.Address, &s.MerchandiserID,
				&s.MeFullName, &s.MePhone, &jsonSchedules,
			); err != nil {
				continue
			}

			// Распаковываем массив байт JSON в массив структур ManufacturerSchedule языка Go
			if len(jsonSchedules) > 0 {
				_ = json.Unmarshal(jsonSchedules, &s.Schedules)
			} else {
				s.Schedules = []ManufacturerSchedule{}
			}

			stores = append(stores, s)
		}

		json.NewEncoder(w).Encode(stores)
	}
}

// AssignMerchandiserToStoreHandler позволяет СВ назначать, менять или убирать МЕ с торговой точки (Пункт 21.2 ТЗ)
func AssignMerchandiserToStoreHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
			return
		}

		var req AssignMeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		// Безопасность: Проверяем, принадлежит ли эта точка именно этому СВ, чтобы исключить подмену ID
		var belongs bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM stores WHERE id_1c = $1 AND supervisor_id = $2)`
		err := db.QueryRow(ctx, checkQuery, req.StoreId1c, user.ID).Scan(&belongs)
		if err != nil || !belongs {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Доступ запрещен. Эта торговая точка не закреплена за вами"}`))
			return
		}

		// Обновляем МЕ на торговой точке (Пункт 21.2: назначить, заменить, убрать)
		updateQuery := `UPDATE stores SET merchandiser_id = $1 WHERE id_1c = $2`
		_, err = db.Exec(ctx, updateQuery, req.MerchandiserID, req.StoreId1c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка при обновлении мерчандайзера в базе данных"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "Мерчандайзер успешно закреплен за торговой точкой"}`))
	}
}
