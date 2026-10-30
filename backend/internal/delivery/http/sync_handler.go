package http

import (
	"encoding/json"
	"net/http"

	// "strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MobileStoreResponse struct {
	Id1c          string   `json:"id_1c"`
	Region        string   `json:"region"`
	City          string   `json:"city"`
	RetailNetwork string   `json:"retail_network"`
	StoreSign     string   `json:"store_sign"`
	StoreFormat   string   `json:"store_format"`
	Address       string   `json:"address"`
	SvName        string   `json:"sv_name"`
	SvPhone       string   `json:"sv_phone"`
	Latitude      *float64 `json:"latitude"` // ДОБАВЛЕНО
	Longitude     *float64 `json:"longitude"`
}

type MobileSkuResponse struct {
	ID             string `json:"id"`
	ManufacturerID int    `json:"manufacturer_id"`
	Name           string `json:"name"`
	Brand          string `json:"brand"`
	Category       string `json:"category"`
}

type MobileMatrixResponse struct {
	SkuID         string `json:"sku_id"`
	RetailNetwork string `json:"retail_network"`
	StoreFormat   string `json:"store_format"`
	IsAvailable   bool   `json:"is_available"`
}

type MobileQuestionResponse struct {
	ID                    int      `json:"id"` // УНИФИКАЦИЯ: Чистый числовой ID для продакшн-стандарта
	ManufacturerID        int      `json:"manufacturer_id"`
	QuestionnaireID       int      `json:"questionnaire_id"`
	Section               string   `json:"section"`
	ParentQuestionID      *int     `json:"parent_question_id"` // УНИФИКАЦИЯ: Чистый указатель на int
	QuestionText          string   `json:"question_text"`
	ResponseType          string   `json:"response_type"`
	AvailableOptions      []string `json:"available_options"`
	SortOrder             int      `json:"sort_order"`
	ConditionBranch       *bool    `json:"condition_branch"`
	ConditionOperator     *string  `json:"condition_operator"`
	ConditionValueText    []string `json:"condition_value_text"`
	ConditionValueNumeric *float64 `json:"condition_value_numeric"`
	AllowGallery          bool     `json:"allow_gallery"`
	DependsOnQuestionID   *int     `json:"depends_on_question_id"`
	DependsOnOperator     *string  `json:"depends_on_operator"`
	DependsOnValue        *string  `json:"depends_on_value"`
}

type MobileManufacturerResponse struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	WorkDays        []int  `json:"work_days"`
	DurationMinutes int    `json:"duration_minutes"`
}

type MobileStoreManufacturerLinkResponse struct {
	StoreID1c      string `json:"store_id_1c"`
	ManufacturerID int    `json:"manufacturer_id"`
}

type MobileHistoricalAnswerResponse struct {
	StoreID1c   string   `json:"store_id_1c"` // <--- ДОБАВЛЕHО ДЛЯ ИЗОЛЯЦИИ ТОЧЕК
	QuestionID  int      `json:"question_id"`
	SkuID       *string  `json:"sku_id"`
	ValueText   *string  `json:"value_text"`
	ValueNumber *float64 `json:"value_number"`
}

type MobileSyncResponse struct {
	Stores                 []MobileStoreResponse                 `json:"stores"`
	Skus                   []MobileSkuResponse                   `json:"skus"`
	Matrices               []MobileMatrixResponse                `json:"matrices"`
	Questions              []MobileQuestionResponse              `json:"questions"`
	Manufacturers          []MobileManufacturerResponse          `json:"manufacturers"`
	StoreManufacturerLinks []MobileStoreManufacturerLinkResponse `json:"store_manufacturer_links"`
	HistoricalAnswers      []MobileHistoricalAnswerResponse      `json:"historical_answers"`
}

func MerchandiserSyncHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка инициализации транзакции синхронизации"}`))
			return
		}
		defer tx.Rollback(ctx)
		storeQuery := `
			SELECT s.id_1c, s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
				COALESCE(sv.full_name, 'Не назначен') AS sv_name, COALESCE(sv.phone, '') AS sv_phone,
				s.latitude, s.longitude
			FROM stores s
			LEFT JOIN users sv ON s.supervisor_id = sv.id AND sv.role = 'supervisor'
			WHERE s.merchandiser_id = $1`

		rows, err := tx.Query(ctx, storeQuery, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка чтения адресной программы из БД"}`))
			return
		}

		storesList := []MobileStoreResponse{}
		for rows.Next() {
			var s MobileStoreResponse
			if err := rows.Scan(&s.Id1c, &s.Region, &s.City, &s.RetailNetwork, &s.StoreSign, &s.StoreFormat, &s.Address, &s.SvName, &s.SvPhone, &s.Latitude, &s.Longitude); err != nil {
				rows.Close()
				return
			}
			storesList = append(storesList, s)
		}
		rows.Close()

		if len(storesList) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MobileSyncResponse{
				Stores:                 []MobileStoreResponse{},
				Skus:                   []MobileSkuResponse{},
				Matrices:               []MobileMatrixResponse{},
				Questions:              []MobileQuestionResponse{},
				Manufacturers:          []MobileManufacturerResponse{},
				StoreManufacturerLinks: []MobileStoreManufacturerLinkResponse{},
			})
			return
		}

		skuQuery := `
			SELECT DISTINCT sk.id, sk.manufacturer_id, sk.name, sk.brand, sk.category
			FROM skus sk
			INNER JOIN manufacturer_stores ms ON sk.manufacturer_id = ms.manufacturer_id
			INNER JOIN stores s ON ms.store_id_1c = s.id_1c
			WHERE s.merchandiser_id = $1`

		skuRows, err := tx.Query(ctx, skuQuery, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		skusList := []MobileSkuResponse{}
		manufacturersMap := make(map[int]bool)
		for skuRows.Next() {
			var sk MobileSkuResponse
			if err := skuRows.Scan(&sk.ID, &sk.ManufacturerID, &sk.Name, &sk.Brand, &sk.Category); err != nil {
				skuRows.Close()
				return
			}
			skusList = append(skusList, sk)
			manufacturersMap[sk.ManufacturerID] = true
		}
		skuRows.Close()

		matrixQuery := `
			SELECT DISTINCT sm.sku_id, sm.retail_network, sm.store_format
			FROM store_matrices sm
			INNER JOIN stores s ON sm.retail_network = s.retail_network AND sm.store_format = s.store_format
			WHERE s.merchandiser_id = $1`

		matrixRows, err := tx.Query(ctx, matrixQuery, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		matricesList := []MobileMatrixResponse{}
		for matrixRows.Next() {
			var m MobileMatrixResponse
			if err := matrixRows.Scan(&m.SkuID, &m.RetailNetwork, &m.StoreFormat); err != nil {
				matrixRows.Close()
				return
			}
			m.IsAvailable = true
			matricesList = append(matricesList, m)
		}
		matrixRows.Close()

		manufacturersQuery := `
			SELECT DISTINCT m.id, m.name, m.work_days, m.duration_minutes
			FROM manufacturers m
			INNER JOIN manufacturer_stores ms ON m.id = ms.manufacturer_id
			INNER JOIN stores s ON ms.store_id_1c = s.id_1c
			WHERE s.merchandiser_id = $1`

		manRows, err := tx.Query(ctx, manufacturersQuery, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		manufacturersList := []MobileManufacturerResponse{}
		for manRows.Next() {
			var m MobileManufacturerResponse
			if err := manRows.Scan(&m.ID, &m.Name, &m.WorkDays, &m.DurationMinutes); err != nil {
				manRows.Close()
				return
			}
			manufacturersList = append(manufacturersList, m)
		}
		manRows.Close()

		linksQuery := `
			SELECT ms.manufacturer_id, ms.store_id_1c
			FROM manufacturer_stores ms
			INNER JOIN stores s ON ms.store_id_1c = s.id_1c
			WHERE s.merchandiser_id = $1`

		linkRows, err := tx.Query(ctx, linksQuery, user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		linksList := []MobileStoreManufacturerLinkResponse{}
		for linkRows.Next() {
			var l MobileStoreManufacturerLinkResponse
			if err := linkRows.Scan(&l.ManufacturerID, &l.StoreID1c); err != nil {
				linkRows.Close()
				return
			}
			linksList = append(linksList, l)
		}
		linkRows.Close()

		questionsList := []MobileQuestionResponse{}
		for mID := range manufacturersMap {
			var currentQuestionnaireID int
			qHeaderQuery := `SELECT id FROM questionnaires WHERE manufacturer_id = $1 ORDER BY version DESC LIMIT 1`
			err = tx.QueryRow(ctx, qHeaderQuery, mID).Scan(&currentQuestionnaireID)
			if err != nil {
				continue
			}

			questionsQuery := `
			SELECT id, questionnaire_id, section, parent_question_id, question_text, response_type,
			       available_options, sort_order, condition_branch, condition_operator,
			       condition_value_text, condition_value_numeric, allow_gallery,
			       depends_on_question_id, depends_on_operator, depends_on_value
			FROM questionnaire_questions
			WHERE questionnaire_id = $1
			ORDER BY sort_order ASC`

			qRows, err := tx.Query(ctx, questionsQuery, currentQuestionnaireID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			for qRows.Next() {
				var q MobileQuestionResponse

				err := qRows.Scan(
					&q.ID, &q.QuestionnaireID, &q.Section, &q.ParentQuestionID, &q.QuestionText, &q.ResponseType,
					&q.AvailableOptions, &q.SortOrder, &q.ConditionBranch, &q.ConditionOperator,
					&q.ConditionValueText, &q.ConditionValueNumeric, &q.AllowGallery,
					&q.DependsOnQuestionID, &q.DependsOnOperator, &q.DependsOnValue,
				)
				if err != nil {
					qRows.Close()
					return
				}

				q.ManufacturerID = mID

				if q.AvailableOptions == nil {
					q.AvailableOptions = []string{}
				}
				if q.ConditionValueText == nil {
					q.ConditionValueText = []string{}
				}

				questionsList = append(questionsList, q)
			}
			qRows.Close()
		}

		historicalAnswersList := []MobileHistoricalAnswerResponse{}

		// Для каждой назначенной мерчандайзеру точки ищем последние ответы по текущим активным анкетам
		for _, storeItem := range storesList {
			for mID := range manufacturersMap {
				var currentQID int
				var isAutoClone bool

				// Проверяем, включен ли владельцем флаг автоподстановки для актуальной версии анкеты бренда
				qErr := tx.QueryRow(ctx, `
					SELECT id, is_auto_clone_enabled 
					FROM questionnaires 
					WHERE manufacturer_id = $1 
					ORDER BY version DESC LIMIT 1`, mID).Scan(&currentQID, &isAutoClone)

				if qErr != nil || !isAutoClone {
					continue
				}

				// ИСПРАВЛЕНО ДЛЯ ДЕПЛОЯ: Ищем последний визит, где БЫЛ заполнен отчет именно по ЭТОМУ производителю mID
				var lastVisitID string
				vErr := tx.QueryRow(ctx, `
					SELECT v.id FROM visits v
					INNER JOIN visit_manufacturer_reports vmr ON v.id = vmr.visit_id
					WHERE v.store_id_1c = $1 AND v.merchandiser_id = $2 AND v.status = 'finished' AND vmr.manufacturer_id = $3
					ORDER BY v.started_at DESC LIMIT 1`, storeItem.Id1c, user.ID, mID).Scan(&lastVisitID)

				if vErr != nil {
					continue
				}

				// Вытаскиваем все текстовые и числовые ответы (фотографии игнорируем)
				ansQuery := `
					SELECT ra.question_id, ra.sku_id, ra.value_text, ra.value_number
					FROM report_answers ra
					INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
					INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
					WHERE vmr.visit_id = $1 AND vmr.questionnaire_id = $2 AND qq.response_type != 'photo'`

				ansRows, err := tx.Query(ctx, ansQuery, lastVisitID, currentQID)
				if err != nil {
					continue
				}

				for ansRows.Next() {
					var ha MobileHistoricalAnswerResponse
					if err := ansRows.Scan(&ha.QuestionID, &ha.SkuID, &ha.ValueText, &ha.ValueNumber); err == nil {
						ha.StoreID1c = storeItem.Id1c // <--- Привязываем жесткий код 1С
						historicalAnswersList = append(historicalAnswersList, ha)
					}
				}

				ansRows.Close()
			}
		}

		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := MobileSyncResponse{
			Stores:                 storesList,
			Skus:                   skusList,
			Matrices:               matricesList,
			Questions:              questionsList,
			Manufacturers:          manufacturersList,
			StoreManufacturerLinks: linksList,
			HistoricalAnswers:      historicalAnswersList,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
