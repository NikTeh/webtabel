package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	// "github.com/jackc/pgx"
	// "github.com/xuri/excelize"
	// g i t h u b . c o m / a w s / a w s - s d k - g o - v 2 / s e r v i c e / s 3
)

// ИСПРАВЛЕНО: Имя структуры приведено к строгому соответствию с циклом сканирования СУБД
type VisitReportJsonRow struct {
	ID            string `json:"id"`
	StoreID       string `json:"store_id"`
	StartedAt     string `json:"started_at"`
	Duration      string `json:"duration"`
	MeName        string `json:"me_name"`
	MePhone       string `json:"me_phone"`
	SvName        string `json:"sv_name"`
	SvPhone       string `json:"sv_phone"`
	Region        string `json:"region"`
	City          string `json:"city"`
	Network       string `json:"network"`
	StoreSign     string `json:"store_sign"`
	StoreFormat   string `json:"store_format"`
	Address       string `json:"address"`
	Manufacturers string `json:"manufacturers"` // Строка брендов через разделитель " | "
	IsCompleted   bool   `json:"is_completed"`
	Coordinates   string `json:"coordinates"`
	IsOnline      bool   `json:"is_online"`
	IsScheduled   bool   `json:"is_scheduled"`
}

// ИСПРАВЛЕНО: Добавлено поле VisitID для точного соответствия со сканированием СУБД и ТЗ экспорта
type AnswerReportJsonRow struct {
	VisitID          string `json:"visit_id"` // Сквозной UUID для красивого однострочного свода
	StoreID          string `json:"store_id"`
	ManufacturerName string `json:"manufacturer_name"`
	StartedAt        string `json:"started_at"`
	Duration         string `json:"duration"`
	MeName           string `json:"me_name"`
	MePhone          string `json:"me_phone"`
	SvName           string `json:"sv_name"`
	SvPhone          string `json:"sv_phone"`
	Region           string `json:"region"`
	City             string `json:"city"`
	Network          string `json:"network"`
	StoreSign        string `json:"store_sign"`
	StoreFormat      string `json:"store_format"`
	Address          string `json:"address"`
	Coordinates      string `json:"coordinates"`
	IsOnline         bool   `json:"is_online"`
	QuestionText     string `json:"question_text"`
	AnswerValue      string `json:"answer_value"`
}

// type PhotoReportJsonRow struct {
// 	ID           int    `json:"id"`
// 	S3Url        string `json:"s3_url"`
// 	S3Key        string `json:"s3_key"`
// 	StoreID      string `json:"store_id"`
// 	Manufacturer string `json:"manufacturer_name"`
// 	StartedAt    string `json:"started_at"`
// 	Duration     string `json:"duration"`
// 	MeName       string `json:"me_name"`
// 	MePhone      string `json:"me_phone"`
// 	SvName       string `json:"sv_name"`
// 	SvPhone      string `json:"sv_phone"`
// 	Region       string `json:"region"`
// 	City         string `json:"city"`
// 	Network      string `json:"network"`
// 	StoreSign    string `json:"store_sign"`
// 	StoreFormat  string `json:"store_format"`
// 	Address      string `json:"address"`
// 	Coordinates  string `json:"coordinates"`
// 	IsOnline     bool   `json:"is_online"`
// }

// Модернизируем структуру ответа PhotoReportJsonRow, добавляя метаданные по ТЗ
type PhotoReportJsonRow struct {
	ID           int    `json:"id"`
	S3Url        string `json:"s3_url"`
	S3Key        string `json:"s3_key"`
	StoreID      string `json:"store_id"`
	Manufacturer string `json:"manufacturer_name"`
	StartedAt    string `json:"started_at"`
	Duration     string `json:"duration"`
	MeName       string `json:"me_name"`
	MePhone      string `json:"me_phone"`
	SvName       string `json:"sv_name"`
	SvPhone      string `json:"sv_phone"`
	Region       string `json:"region"`
	City         string `json:"city"`
	Network      string `json:"network"`
	StoreSign    string `json:"store_sign"`
	StoreFormat  string `json:"store_format"`
	Address      string `json:"address"`
	Coordinates  string `json:"coordinates"`
	IsOnline     bool   `json:"is_online"`
	QuestionText string `json:"question_text"` // ДОБАВЛЕНО
	SkuName      string `json:"sku_name"`      // ДОБАВЛЕНО
	BrandName    string `json:"brand_name"`    // ДОБАВЛЕНО
}

// GetReportRegionsHandler возвращает живой список регионов агентства из БД
func GetReportRegionsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		rows, err := db.Query(r.Context(), "SELECT DISTINCT region FROM stores WHERE region != '' ORDER BY region ASC")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		list := []string{}
		for rows.Next() {
			var reg string
			if err := rows.Scan(&reg); err == nil {
				list = append(list, reg)
			}
		}
		json.NewEncoder(w).Encode(list)
	}
}

// GetReportSupervisorsHandler возвращает живой список СВ из БД
func GetReportSupervisorsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		rows, err := db.Query(r.Context(), "SELECT id, full_name FROM users WHERE role = 'supervisor' ORDER BY full_name ASC")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		list := []map[string]interface{}{}
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err == nil {
				list = append(list, map[string]interface{}{"id": id, "full_name": name})
			}
		}
		json.NewEncoder(w).Encode(list)
	}
}

// GetVisitsReportHandler извлекает визиты со склеенными брендами через string_agg
func GetVisitsReportHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		user, _ := UserFromContext(ctx)
		q := r.URL.Query()

		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
		region, svIDStr, mIDStr := q.Get("region"), q.Get("supervisor_id"), q.Get("manufacturer_id")

		if user.Role == "manufacturer" && user.ManufacturerID != nil {
			mIDStr = strconv.Itoa(*user.ManufacturerID)
		}

		sqlQuery := `
			SELECT
				v.id::text, v.store_id_1c, v.started_at, COALESCE(v.ended_at, v.started_at),
				me.full_name, me.phone, COALESCE(sv.full_name, 'Не назначен'), COALESCE(sv.phone, ''),
				s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
				v.is_online, v.is_scheduled, COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'),
				COALESCE(string_agg(DISTINCT m.name, ' | '), 'Нет контрактов') AS manufacturers,
				EXISTS(
					SELECT 1 FROM report_answers ra
					INNER JOIN visit_manufacturer_reports vmr2 ON ra.report_id = vmr2.id
					WHERE vmr2.visit_id = v.id AND (ra.value_text IS NOT NULL OR ra.value_number IS NOT NULL)
				) AS is_completed
			FROM visits v
			INNER JOIN stores s ON v.store_id_1c = s.id_1c
			INNER JOIN users me ON v.merchandiser_id = me.id
			LEFT JOIN users sv ON s.supervisor_id = sv.id
			LEFT JOIN visit_manufacturer_reports vmr ON v.id = vmr.visit_id
			LEFT JOIN manufacturers m ON vmr.manufacturer_id = m.id
			WHERE v.started_at::date BETWEEN $1::date AND $2::date`

		args := []interface{}{dateFrom, dateTo}
		pIdx := 3
		if mIDStr != "" {
			sqlQuery += fmt.Sprintf(" AND vmr.manufacturer_id = $%d", pIdx)
			args = append(args, mIDStr)
			pIdx++
		}
		if region != "" {
			sqlQuery += fmt.Sprintf(" AND s.region = $%d", pIdx)
			args = append(args, region)
			pIdx++
		}
		if svIDStr != "" {
			sqlQuery += fmt.Sprintf(" AND s.supervisor_id = $%d", pIdx)
			args = append(args, svIDStr)
			pIdx++
		}

		sqlQuery += ` GROUP BY v.id, v.store_id_1c, v.started_at, v.ended_at, me.full_name, me.phone, sv.full_name, sv.phone,
		                       s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address, v.is_online, v.is_scheduled, v.start_latitude, v.start_longitude
		             ORDER BY v.started_at DESC`

		rows, err := db.Query(ctx, sqlQuery, args...)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		result := []VisitReportJsonRow{}
		for rows.Next() {
			var r VisitReportJsonRow
			var start, end time.Time
			_ = rows.Scan(
				&r.ID, &r.StoreID, &start, &end, &r.MeName, &r.MePhone, &r.SvName, &r.SvPhone,
				&r.Region, &r.City, &r.Network, &r.StoreSign, &r.StoreFormat, &r.Address,
				&r.IsOnline, &r.IsScheduled, &r.Coordinates, &r.Manufacturers, &r.IsCompleted,
			)
			r.StartedAt = start.Format("2006-01-02 15:04")
			r.Duration = end.Sub(start).Round(time.Minute).String()

			if user.Role == "supervisor" && r.SvName != user.FullName {
				continue
			}
			result = append(result, r)
		}
		json.NewEncoder(w).Encode(result)
	}
}

// GetAnswersReportHandler выводит анкеты с привязкой к visit_id в качестве главного ключа
func GetAnswersReportHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		user, _ := UserFromContext(ctx)
		q := r.URL.Query()

		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
		mIDStr := q.Get("manufacturer_id")

		if user.Role == "manufacturer" && user.ManufacturerID != nil {
			mIDStr = strconv.Itoa(*user.ManufacturerID)
		}

		sqlQuery := `
			SELECT 
				v.id::text, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at), me.full_name, me.phone,
				COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''), s.region, s.city, s.retail_network,
				s.store_sign, s.store_format, s.address, COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'),
				v.is_online, qq.question_text, COALESCE(ra.value_text, ra.value_number::text)
			FROM report_answers ra
			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
			INNER JOIN visits v ON vmr.visit_id = v.id
			INNER JOIN stores s ON v.store_id_1c = s.id_1c
			INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
			INNER JOIN users me ON v.merchandiser_id = me.id
			INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
			LEFT JOIN users sv ON s.supervisor_id = sv.id
			WHERE vmr.manufacturer_id = $1 AND v.started_at::date BETWEEN $2::date AND $3::date
			ORDER BY v.started_at DESC, ra.id ASC`

		rows, err := db.Query(ctx, sqlQuery, mIDStr, dateFrom, dateTo)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		result := []AnswerReportJsonRow{}
		for rows.Next() {
			var r AnswerReportJsonRow
			var start, end time.Time
			_ = rows.Scan(
				&r.VisitID, &r.StoreID, &r.ManufacturerName, &start, &end, &r.MeName, &r.MePhone,
				&r.SvName, &r.SvPhone, &r.Region, &r.City, &r.Network, &r.StoreSign,
				&r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline, &r.QuestionText, &r.AnswerValue,
			)
			r.StartedAt = start.Format("2006-01-02 15:04")
			r.Duration = end.Sub(start).Round(time.Minute).String()

			if user.Role == "supervisor" && r.SvName != user.FullName {
				continue
			}
			result = append(result, r)
		}
		json.NewEncoder(w).Encode(result)
	}
}

// // GetPhotosReportHandler возвращает JSON-список медиафайлов S3 с метаданными визитов (Восстановлено)
// func GetPhotosReportHandler(db *pgxpool.Pool) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		ctx := r.Context()
// 		user, _ := UserFromContext(ctx)
// 		q := r.URL.Query()

// 		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
// 		mIDStr := q.Get("manufacturer_id")

// 		if user.Role == "manufacturer" && user.ManufacturerID != nil {
// 			mIDStr = strconv.Itoa(*user.ManufacturerID)
// 		}

// 		sqlQuery := `
// 			SELECT
// 				rp.id, rp.s3_url, rp.s3_key, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at),
// 				me.full_name, me.phone, COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''),
// 				s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
// 				COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'), v.is_online
// 			FROM report_photos rp
// 			INNER JOIN report_answers ra ON rp.answer_id = ra.id
// 			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
// 			INNER JOIN visits v ON vmr.visit_id = v.id
// 			INNER JOIN stores s ON v.store_id_1c = s.id_1c
// 			INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
// 			INNER JOIN users me ON v.merchandiser_id = me.id
// 			LEFT JOIN users sv ON s.supervisor_id = sv.id
// 			WHERE v.started_at::date BETWEEN $1::date AND $2::date`

// 		args := []interface{}{dateFrom, dateTo}
// 		if mIDStr != "" {
// 			sqlQuery += " AND vmr.manufacturer_id = $3"
// 			args = append(args, mIDStr)
// 		}

// 		sqlQuery += " ORDER BY rp.id DESC"
// 		rows, err := db.Query(ctx, sqlQuery, args...)
// 		if err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}
// 		defer rows.Close()

// 		result := []PhotoReportJsonRow{}
// 		for rows.Next() {
// 			var r PhotoReportJsonRow
// 			var start, end time.Time
// 			_ = rows.Scan(
// 				&r.ID, &r.S3Url, &r.S3Key, &r.StoreID, &r.Manufacturer, &start, &end,
// 				&r.MeName, &r.MePhone, &r.SvName, &r.SvPhone, &r.Region, &r.City,
// 				&r.Network, &r.StoreSign, &r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline,
// 			)
// 			r.StartedAt = start.Format("2006-01-02 15:04")
// 			r.Duration = end.Sub(start).Round(time.Minute).String()

// 			if user.Role == "supervisor" && r.SvName != user.FullName {
// 				continue
// 			}
// 			result = append(result, r)
// 		}
// 		json.NewEncoder(w).Encode(result)
// 	}
// }

// // Переписываем хэндлер получения фотографий с поддержкой LIMIT/OFFSET пагинации
// func GetPhotosReportHandler(db *pgxpool.Pool) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		ctx := r.Context()
// 		user, _ := UserFromContext(ctx)
// 		q := r.URL.Query()

// 		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
// 		mIDStr := q.Get("manufacturer_id")
// 		pageStr, limitStr := q.Get("page"), q.Get("limit")

// 		page, errPage := strconv.Atoi(pageStr)
// 		limit, errLimit := strconv.Atoi(limitStr)
// 		if errPage != nil || page <= 0 {
// 			page = 1
// 		}
// 		if errLimit != nil || limit <= 0 {
// 			limit = 20
// 		}
// 		offset := (page - 1) * limit

// 		if user.Role == "manufacturer" && user.ManufacturerID != nil {
// 			mIDStr = strconv.Itoa(*user.ManufacturerID)
// 		}

// 		// sqlQuery := `
// 		// 	SELECT
// 		// 		rp.id, rp.s3_url, rp.s3_key, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at),
// 		// 		me.full_name, me.phone, COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''),
// 		// 		s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
// 		// 		COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'), v.is_online,
// 		// 		qq.question_text, COALESCE(sk.name, 'Общий вопрос'), COALESCE(sk.brand, '—')
// 		// 	FROM report_photos rp
// 		// 	INNER JOIN report_answers ra ON rp.answer_id = ra.id
// 		// 	INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
// 		// 	INNER JOIN visits v ON vmr.visit_id = v.id
// 		// 	INNER JOIN stores s ON v.store_id_1c = s.id_1c
// 		// 	INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
// 		// 	INNER JOIN users me ON v.merchandiser_id = me.id
// 		// 	INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
// 		// 	LEFT JOIN skus sk ON ra.sku_id = sk.id
// 		// 	LEFT JOIN users sv ON s.supervisor_id = sv.id
// 		// 	WHERE v.started_at::date BETWEEN $1::date AND $2::date`

// 		// ИСПРАВЛЕНО: Связываем таблицы по s3_key, так как ra.value_text содержит точный путь к фото
// 		sqlQuery := `
// 			SELECT
// 				rp.id, rp.s3_url, rp.s3_key, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at),
// 				me.full_name, me.phone, COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''),
// 				s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
// 				COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'), v.is_online,
// 				qq.question_text, COALESCE(sk.name, 'Общий вопрос'), COALESCE(sk.brand, '—')
// 			FROM report_photos rp
// 			INNER JOIN report_answers ra ON ra.value_text LIKE '%' || rp.s3_key || '%'
// 			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
// 			INNER JOIN visits v ON vmr.visit_id = v.id
// 			INNER JOIN stores s ON v.store_id_1c = s.id_1c
// 			INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
// 			INNER JOIN users me ON v.merchandiser_id = me.id
// 			INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
// 			LEFT JOIN skus sk ON ra.sku_id = sk.id
// 			LEFT JOIN users sv ON s.supervisor_id = sv.id
// 			WHERE v.started_at::date BETWEEN $1::date AND $2::date`

// 		args := []interface{}{dateFrom, dateTo}
// 		pIdx := 3
// 		if mIDStr != "" {
// 			sqlQuery += fmt.Sprintf(" AND vmr.manufacturer_id = $%d", pIdx)
// 			args = append(args, mIDStr)
// 			pIdx++
// 		}

// 		sqlQuery += fmt.Sprintf(" ORDER BY rp.id DESC LIMIT $%d OFFSET $%d", pIdx, pIdx+1)
// 		args = append(args, limit, offset)

// 		rows, err := db.Query(ctx, sqlQuery, args...)
// 		if err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}
// 		defer rows.Close()

// 		//start
// 		// Инициализируем S3 Presign Клиент перед циклом сканирования строк СУБД
// 		// Используем официальные системные пакеты S3 SDK v2
// 		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
// 		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
// 		bucketName := os.Getenv("S3_BUCKET_NAME")

// 		s3Client := s3.New(s3.Options{
// 			Region: "ru-1",
// 			// BaseEndpoint: aws.String("https://twcstorage.ru"),
// 			BaseEndpoint: aws.String("https://s3.twcstorage.ru"),
// 			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
// 				accessKey, secretKey, "",
// 			)),
// 		})

// 		// Создаем менеджер подписи ссылок
// 		presignClient := s3.NewPresignClient(s3Client)

// 		//end

// 		result := []PhotoReportJsonRow{}
// 		// for rows.Next() {
// 		// 	var r PhotoReportJsonRow
// 		// 	var start, end time.Time
// 		// 	_ = rows.Scan(
// 		// 		&r.ID, &r.S3Url, &r.S3Key, &r.StoreID, &r.Manufacturer, &start, &end,
// 		// 		&r.MeName, &r.MePhone, &r.SvName, &r.SvPhone, &r.Region, &r.City,
// 		// 		&r.Network, &r.StoreSign, &r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline,
// 		// 		&r.QuestionText, &r.SkuName, &r.BrandName,
// 		// 	)
// 		// 	r.StartedAt = start.Format("2006-01-02 15:04")
// 		// 	r.Duration = end.Sub(start).Round(time.Minute).String()

// 		// 	if user.Role == "supervisor" && r.SvName != user.FullName {
// 		// 		continue
// 		// 	}
// 		// 	result = append(result, r)
// 		// }
// 		for rows.Next() {
// 			var r PhotoReportJsonRow
// 			var start, end time.Time
// 			var rawDbUrl string // Прослойка для считывания статического URL из БД

// 			_ = rows.Scan(
// 				&r.ID, &rawDbUrl, &r.S3Key, &r.StoreID, &r.Manufacturer, &start, &end,
// 				&r.MeName, &r.MePhone, &r.SvName, &r.SvPhone, &r.Region, &r.City,
// 				&r.Network, &r.StoreSign, &r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline,
// 				&r.QuestionText, &r.SkuName, &r.BrandName,
// 			)
// 			r.StartedAt = start.Format("2006-01-02 15:04")
// 			r.Duration = end.Sub(start).Round(time.Minute).String()

// 			if user.Role == "supervisor" && r.SvName != user.FullName {
// 				continue
// 			}

// 			// ГЕНЕРАЦИЯ ОДНОРАЗОВОЙ ССЫЛКИ ДЛЯ ПРИВАТНОГО БАКЕТА ТАЙМВЕБ:
// 			// Ссылка будет валидна ровно 15 минут, чего с запасом хватает на сессию инспекции ЛК
// 			presignedReq, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
// 				Bucket: aws.String(bucketName),
// 				Key:    aws.String(r.S3Key), // Подставляем "visits/UUID_ВИЗИТА/..."
// 			}, s3.WithPresignExpires(15*time.Minute))

// 			if err == nil {
// 				// Заменяем заблокированный статический URL на живой подписанный токен доступа S3
// 				r.S3Url = presignedReq.URL
// 			} else {
// 				// Резервный дефолт на случай сбоя API S3
// 				r.S3Url = rawDbUrl
// 			}

// 			result = append(result, r)
// 		}
// 		json.NewEncoder(w).Encode(result)
// 	}
// }
//todo не удалять это для отладки
// func GetPhotosReportHandler(db *pgxpool.Pool) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		ctx := r.Context()
// 		user, _ := UserFromContext(ctx)
// 		q := r.URL.Query()

// 		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
// 		mIDStr := q.Get("manufacturer_id")
// 		pageStr, limitStr := q.Get("page"), q.Get("limit")

// 		page, _ := strconv.Atoi(pageStr)
// 		limit, _ := strconv.Atoi(limitStr)
// 		if page <= 0 {
// 			page = 1
// 		}
// 		if limit <= 0 {
// 			limit = 20
// 		}
// 		offset := (page - 1) * limit

// 		if user.Role == "manufacturer" && user.ManufacturerID != nil {
// 			mIDStr = strconv.Itoa(*user.ManufacturerID)
// 		}

// 		// ЛОГ 1: Выводим входящие параметры фильтрации из React
// 		fmt.Printf("\n========== [СТАРТ ПОИСКА ФОТО] ==========\n")
// 		fmt.Printf("Входящие фильтры -> DateFrom: %s, DateTo: %s, ManufacturerID: %s\n", dateFrom, dateTo, mIDStr)
// 		fmt.Printf("Пагинация -> Page: %d, Limit: %d, Offset: %d\n", page, limit, offset)

// 		sqlQuery := `
// 			SELECT
// 				rp.id, rp.s3_url, rp.s3_key, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at),
// 				me.full_name, me.phone, COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''),
// 				s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
// 				COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'), v.is_online,
// 				qq.question_text, COALESCE(sk.name, 'Общий вопрос'), COALESCE(sk.brand, '—')
// 			FROM report_photos rp
// 			INNER JOIN report_answers ra ON ra.value_text LIKE '%' || rp.s3_key || '%'
// 			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
// 			INNER JOIN visits v ON vmr.visit_id = v.id
// 			INNER JOIN stores s ON v.store_id_1c = s.id_1c
// 			INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
// 			INNER JOIN users me ON v.merchandiser_id = me.id
// 			INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
// 			LEFT JOIN skus sk ON ra.sku_id = sk.id
// 			LEFT JOIN users sv ON s.supervisor_id = sv.id
// 			WHERE v.started_at::date BETWEEN $1::date AND $2::date`

// 		args := []interface{}{dateFrom, dateTo}
// 		pIdx := 3
// 		if mIDStr != "" {
// 			sqlQuery += fmt.Sprintf(" AND vmr.manufacturer_id::text = $%d", pIdx)
// 			args = append(args, mIDStr)
// 			pIdx++
// 		}

// 		sqlQuery += fmt.Sprintf(" ORDER BY rp.id DESC LIMIT $%d OFFSET $%d", pIdx, pIdx+1)
// 		args = append(args, limit, offset)

// 		// ЛОГ 2: Выводим финальные аргументы, передаваемые драйверу pgx
// 		fmt.Printf("Аргументы SQL-запроса: %v\n", args)

// 		rows, err := db.Query(ctx, sqlQuery, args...)
// 		if err != nil {
// 			fmt.Printf("🛑 КРИТИЧЕСКАЯ ОШИБКА ВЫПОЛНЕНИЯ ЗАПРОСА В PG: %v\n", err)
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}
// 		defer rows.Close()

// 		bucketName := os.Getenv("S3_BUCKET_NAME")
// 		result := []PhotoReportJsonRow{}

// 		rowCount := 0
// 		for rows.Next() {
// 			rowCount++
// 			var r PhotoReportJsonRow
// 			var start, end time.Time
// 			var rawDbUrl string

// 			errScan := rows.Scan(
// 				&r.ID, &rawDbUrl, &r.S3Key, &r.StoreID, &r.Manufacturer, &start, &end,
// 				&r.MeName, &r.MePhone, &r.SvName, &r.SvPhone, &r.Region, &r.City,
// 				&r.Network, &r.StoreSign, &r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline,
// 				&r.QuestionText, &r.SkuName, &r.BrandName,
// 			)
// 			if errScan != nil {
// 				fmt.Printf("⚠️ Ошибка сканирования строки (Scan): %v\n", errScan)
// 				continue
// 			}

// 			r.StartedAt = start.Format("2006-01-02 15:04")
// 			r.Duration = end.Sub(start).Round(time.Minute).String()

// 			if user.Role == "supervisor" && r.SvName != user.FullName {
// 				continue
// 			}

// 			// ЛОГ 3: Выводим параметры запроса к API Timeweb Cloud для генерации Presigned URL
// 			fmt.Printf("--- Найдена строка [%d] в БД ---\n", rowCount)
// 			fmt.Printf("  Бакет S3: %s\n", bucketName)
// 			fmt.Printf("  S3 Key (запрос на Timeweb): %s\n", r.S3Key)
// 			fmt.Printf("  Привязка к точке: %s, Вопрос: %s\n", r.StoreID, r.QuestionText)

// 			result = append(result, r)
// 		}

// 		// ЛОГ 4: Итоговый размер сформированного ответа
// 		fmt.Printf("Всего строк извлечено из БД и прошло фильтрацию ролей: %d\n", len(result))
// 		fmt.Printf("=========================================\n")

// 		json.NewEncoder(w).Encode(result)
// 	}
// }

// GetPhotosReportHandler возвращает порционный JSON-реестр полевых медиафайлов с подписью ссылок для приватного S3
func GetPhotosReportHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		user, _ := UserFromContext(ctx)
		q := r.URL.Query()

		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
		mIDStr := q.Get("manufacturer_id")
		pageStr, limitStr := q.Get("page"), q.Get("limit")

		page, _ := strconv.Atoi(pageStr)
		limit, _ := strconv.Atoi(limitStr)
		if page <= 0 {
			page = 1
		}
		if limit <= 0 {
			limit = 20
		}
		offset := (page - 1) * limit

		if user.Role == "manufacturer" && user.ManufacturerID != nil {
			mIDStr = strconv.Itoa(*user.ManufacturerID)
		}

		// ЛОГ 1: Выводим входящие параметры фильтрации из React
		// fmt.Printf("\n========== [СТАРТ ПОИСКА ФОТО] ==========\n")
		// fmt.Printf("Входящие фильтры -> DateFrom: %s, DateTo: %s, ManufacturerID: %s\n", dateFrom, dateTo, mIDStr)
		// fmt.Printf("Пагинация -> Page: %d, Limit: %d, Offset: %d\n", page, limit, offset)

		// ИСПРАВЛЕНО: Чистый реляционный JOIN по числовым внешним ключам
		// ИСПРАВЛЕНО: Граница дат сдвинута на +1 день для захвата вечерней выкладки товаров
		sqlQuery := `
			SELECT 
				rp.id, rp.s3_url, rp.s3_key, v.store_id_1c, m.name, v.started_at, COALESCE(v.ended_at, v.started_at),
				me.full_name, me.phone, COALESCE(sv.full_name, '—'), COALESCE(sv.phone, ''),
				s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address,
				COALESCE(v.start_latitude::text || ', ' || v.start_longitude::text, '0, 0'), v.is_online,
				qq.question_text, COALESCE(sk.name, 'Общий вопрос'), COALESCE(sk.brand, '—')
			FROM report_photos rp
			INNER JOIN report_answers ra ON rp.answer_id = ra.id
			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
			INNER JOIN visits v ON vmr.visit_id = v.id
			INNER JOIN stores s ON v.store_id_1c = s.id_1c
			INNER JOIN manufacturers m ON vmr.manufacturer_id = m.id
			INNER JOIN users me ON v.merchandiser_id = me.id
			INNER JOIN questionnaire_questions qq ON ra.question_id = qq.id
			LEFT JOIN skus sk ON ra.sku_id = sk.id
			LEFT JOIN users sv ON s.supervisor_id = sv.id
			WHERE v.started_at >= $1::timestamp 
			  AND v.started_at < $2::date + interval '1 day'`

		args := []interface{}{dateFrom, dateTo}
		pIdx := 3
		if mIDStr != "" {
			sqlQuery += fmt.Sprintf(" AND vmr.manufacturer_id::text = $%d", pIdx)
			args = append(args, mIDStr)
			pIdx++
		}

		sqlQuery += fmt.Sprintf(" ORDER BY rp.id DESC LIMIT $%d OFFSET $%d", pIdx, pIdx+1)
		args = append(args, limit, offset)

		// ЛОГ 2: Выводим финальные аргументы, передаваемые драйверу pgx
		// fmt.Printf("Аргументы SQL-запроса: %v\n", args)

		rows, err := db.Query(ctx, sqlQuery, args...)
		if err != nil {
			// fmt.Printf("🛑 КРИТИЧЕСКАЯ ОШИБКА ВЫПОЛНЕНИЯ ЗАПРОСА В PG: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Подключаем S3-клиент подписи приватных ссылок Timeweb Cloud
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		bucketName := os.Getenv("S3_BUCKET_NAME")

		s3Client := s3.New(s3.Options{
			Region:       "ru-1",
			BaseEndpoint: aws.String("https://s3.twcstorage.ru"),
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
				accessKey, secretKey, "",
			)),
		})
		presignClient := s3.NewPresignClient(s3Client)

		result := []PhotoReportJsonRow{}
		rowCount := 0

		for rows.Next() {
			rowCount++
			var r PhotoReportJsonRow
			var start, end time.Time
			var rawDbUrl string

			errScan := rows.Scan(
				&r.ID, &rawDbUrl, &r.S3Key, &r.StoreID, &r.Manufacturer, &start, &end,
				&r.MeName, &r.MePhone, &r.SvName, &r.SvPhone, &r.Region, &r.City,
				&r.Network, &r.StoreSign, &r.StoreFormat, &r.Address, &r.Coordinates, &r.IsOnline,
				&r.QuestionText, &r.SkuName, &r.BrandName,
			)
			if errScan != nil {
				// fmt.Printf("⚠️ Ошибка сканирования строки (Scan): %v\n", errScan)
				continue
			}

			r.StartedAt = start.Format("2006-01-02 15:04")
			r.Duration = end.Sub(start).Round(time.Minute).String()

			if user.Role == "supervisor" && r.SvName != user.FullName {
				continue
			}

			// Безопасное динамическое подписание одноразовой ссылки S3 (валидна 15 минут)
			presignedReq, errPresign := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(r.S3Key),
			}, s3.WithPresignExpires(15*time.Minute))

			if errPresign == nil {
				r.S3Url = presignedReq.URL
			} else {
				r.S3Url = rawDbUrl
			}

			// ЛОГ 3: Выводим параметры запроса к API Timeweb Cloud для генерации Presigned URL
			// fmt.Printf("--- Найдена строка [%d] в БД ---\n", rowCount)
			// fmt.Printf("  Бакет S3: %s\n", bucketName)
			// fmt.Printf("  S3 Key (запрос на Timeweb): %s\n", r.S3Key)
			// fmt.Printf("  Привязка к точке: %s, Вопрос: %s\n", r.StoreID, r.QuestionText)

			result = append(result, r)
		}

		// ЛОГ 4: Итоговый размер сформированного ответа
		// fmt.Printf("Всего строк извлечено из БД и прошло фильтрацию ролей: %d\n", len(result))
		// fmt.Printf("=========================================\n")

		json.NewEncoder(w).Encode(result)
	}
}

// GetMapConfigHandler считывает коммерческий токен Яндекса из .env и отдает его фронтенду
func GetMapConfigHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Читаем ключ из операционной системы, куда его загрузил метод loadEnv() из main.go
		apiKey := os.Getenv("YANDEX_MAPS_API_KEY")
		if apiKey == "" {
			// Если ключ в .env не задан, отдаем дефолтный тестовый токен
			apiKey = "ef4c2744-c6bf-42e7-8b93-e1e924d9f86e"
		}

		resp := map[string]string{
			"api_key": apiKey,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// GetReportMerchandisersHandler реализует живой поиск по 1500 МЕ с лимитом для защиты памяти
func GetReportMerchandisersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		search := r.URL.Query().Get("search")

		if len(search) < 2 {
			// Не мучаем базу данных, если введено меньше 2 символов
			json.NewEncoder(w).Encode([]string{})
			return
		}

		// Выборка строго по роли merchandiser с защитным лимитом в 15 строк
		query := `
			SELECT full_name 
			FROM users 
			WHERE role = 'merchandiser' AND full_name ILIKE $1 
			ORDER BY full_name ASC 
			LIMIT 15`

		rows, err := db.Query(ctx, query, "%"+search+"%")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		list := []string{}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				list = append(list, name)
			}
		}
		json.NewEncoder(w).Encode(list)
	}
}
