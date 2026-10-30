package http

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
	// "github.com/aws/aws-sdk-go-v2"
	// "github.com/aws/aws-sdk-go-v2/config"
)

// ExportVisitsXlsxHandler формирует выгрузку посещений со всеми новыми столбцами
func ExportVisitsXlsxHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		queryValues := r.URL.Query()

		region := queryValues.Get("region")
		svIDStr := queryValues.Get("supervisor_id")
		mIDStr := queryValues.Get("manufacturer_id")
		dateFrom := queryValues.Get("date_from")
		dateTo := queryValues.Get("date_to")

		// Рабочий SQL-запрос, полностью идентичный экранному обработчику
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
		placeholderIdx := 3

		if mIDStr != "" {
			sqlQuery += fmt.Sprintf(" AND vmr.manufacturer_id::text = $%d", placeholderIdx)
			args = append(args, mIDStr)
			placeholderIdx++
		}
		if region != "" {
			sqlQuery += fmt.Sprintf(" AND s.region = $%d", placeholderIdx)
			args = append(args, region)
			placeholderIdx++
		}
		if svIDStr != "" {
			sqlQuery += fmt.Sprintf(" AND s.supervisor_id::text = $%d", placeholderIdx)
			args = append(args, svIDStr)
			placeholderIdx++
		}

		sqlQuery += ` GROUP BY v.id, v.store_id_1c, v.started_at, v.ended_at, me.full_name, me.phone, sv.full_name, sv.phone,
		                       s.region, s.city, s.retail_network, s.store_sign, s.store_format, s.address, v.is_online, v.is_scheduled, v.start_latitude, v.start_longitude
		             ORDER BY v.started_at DESC`

		rows, err := db.Query(ctx, sqlQuery, args...)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка извлечения строк посещений из PostgreSQL"}`))
			return
		}
		defer rows.Close()
		f := excelize.NewFile()
		sheetName := "Отчеты мерчандайзеров"
		f.SetSheetName("Sheet1", sheetName)

		headers := []string{"ID Точки", "Начало визита", "Длительность", "МЕ ФИО", "МЕ Телефон", "СВ ФИО", "СВ Телефон", "Регион", "Город", "Сеть", "Вывеска", "Формат", "Адрес", "Название Производителей", "Заполнен", "Геокоординаты", "Режим (Онлайн/Оффлайн)", "По маршруту (Да/Нет)"}
		for colIdx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}

		rowIdx := 2
		for rows.Next() {
			var visitID, storeID, meName, mePhone, svName, svPhone, region, city, network, sign, format, address, coords, manufacturers string
			var start, end time.Time
			var isOnline, isScheduled, isCompleted bool

			_ = rows.Scan(
				&visitID, &storeID, &start, &end, &meName, &mePhone, &svName, &svPhone,
				&region, &city, &network, &sign, &format, &address, &isOnline, &isScheduled, &coords, &manufacturers, &isCompleted,
			)

			duration := end.Sub(start).Round(time.Minute).String()
			onlineStr := "Онлайн"
			if !isOnline {
				onlineStr = "Оффлайн"
			}
			schedStr := "План"
			if !isScheduled {
				schedStr = "Внеплан"
			}
			compStr := "Нет"
			if isCompleted {
				compStr = "Да"
			}

			values := []interface{}{
				storeID, start.Format("2006-01-02 15:04"), duration, meName, mePhone, svName, svPhone,
				region, city, network, sign, format, address, manufacturers, compStr, coords, onlineStr, schedStr,
			}
			for colIdx, val := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx)
				f.SetCellValue(sheetName, cell, val)
			}
			rowIdx++
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=visits_report.xlsx")
		w.WriteHeader(http.StatusOK)
		_ = f.Write(w)
	}
}

// ExportAnswersXlsxHandler формирует выгрузку ответов анкет по посимвольно скопированному SQL-запросу
func ExportAnswersXlsxHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		queryValues := r.URL.Query()

		dateFrom := queryValues.Get("date_from")
		dateTo := queryValues.Get("date_to")
		mIDStr := queryValues.Get("manufacturer_id")

		if mIDStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Параметр manufacturer_id обязателен"}`))
			return
		}
		// ИСПРАВЛЕНО: Запрос скопирован один в один из GetAnswersReportHandler, vmr.manufacturer_id::text = $1
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
			WHERE vmr.manufacturer_id::text = $1 AND v.started_at::date BETWEEN $2::date AND $3::date
			ORDER BY v.started_at DESC, ra.id ASC`

		rows, err := db.Query(ctx, sqlQuery, mIDStr, dateFrom, dateTo)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка извлечения анкетных строк из PostgreSQL"}`))
			return
		}
		defer rows.Close()

		f := excelize.NewFile()
		sheetName := "Отчеты мерчандайзеров"
		f.SetSheetName("Sheet1", sheetName)

		headers := []string{"Visit UUID (Сводный ключ)", "ID Точки", "Производитель", "Дата визита", "МЕ ФИО", "Торговая сеть / Адрес", "Вопрос анкеты", "Ответ мерчандайзера"}
		for colIdx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}

		rowIdx := 2
		for rows.Next() {
			var visitID, storeID, mName, meName, mePhone, svName, svPhone, region, city, network, sign, format, address, coords, qText, answerVal string
			var start, end time.Time
			var isOnline bool

			_ = rows.Scan(
				&visitID, &storeID, &mName, &start, &end, &meName, &mePhone,
				&svName, &svPhone, &region, &city, &network, &sign, &format, &address, &coords, &isOnline,
				&qText, &answerVal,
			)

			values := []interface{}{
				visitID, storeID, mName, start.Format("2006-01-02 15:04"), meName,
				network + " — " + address, qText, answerVal,
			}
			for colIdx, val := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx)
				f.SetCellValue(sheetName, cell, val)
			}
			rowIdx++
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=answers_report.xlsx")
		w.WriteHeader(http.StatusOK)
		_ = f.Write(w)
	}
}

// ExportPhotosZipHandler выполняет стриминг zip-архива изображений напрямую в HTTP-ответ
func ExportPhotosZipHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()

		dateFrom, dateTo := q.Get("date_from"), q.Get("date_to")
		mIDStr := q.Get("manufacturer_id")

		if mIDStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Параметр manufacturer_id обязателен для выгрузки ZIP"}`))
			return
		}

		// // Выборка ключей s3_key по ТЗ
		// sqlQuery := `
		// 	SELECT rp.s3_key
		// 	FROM report_photos rp
		// 	INNER JOIN report_answers ra ON rp.answer_id = ra.id
		// 	INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
		// 	INNER JOIN visits v ON vmr.visit_id = v.id
		// 	WHERE vmr.manufacturer_id::text = $1 AND v.started_at::date BETWEEN $2::date AND $3::date`

		// ИСПРАВЛЕНО: Синхронизируем связь по s3_key с логикой текстовых ответов
		sqlQuery := `
			SELECT rp.s3_key 
			FROM report_photos rp
			INNER JOIN report_answers ra ON ra.value_text LIKE '%' || rp.s3_key || '%'
			INNER JOIN visit_manufacturer_reports vmr ON ra.report_id = vmr.id
			INNER JOIN visits v ON vmr.visit_id = v.id
			WHERE vmr.manufacturer_id::text = $1 AND v.started_at::date BETWEEN $2::date AND $3::date`

		rows, err := db.Query(ctx, sqlQuery, mIDStr, dateFrom, dateTo)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		s3Keys := []string{}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err == nil {
				s3Keys = append(s3Keys, key)
			}
		}

		// Если снимков нет, прерываем процедуру
		if len(s3Keys) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Фотоотчеты за выбранный период отсутствуют"}`))
			return
		}

		// Установка правильных streaming заголовков для zip
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=photos_export.zip")
		w.WriteHeader(http.StatusOK)

		// На лету силами archive/zip заворачиваем HTTP-ответ
		zipWriter := zip.NewWriter(w)
		defer zipWriter.Close()

		// s3Client := s3.New(s3.Options{
		// 	Region:       "ru-1",
		// 	BaseEndpoint: aws.String("https://s3.twcstorage.ru"),
		// 	Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		// 		// os.Getenv("AWS_ACCESS_KEY_ID"),
		// 		// os.Getenv("AWS_SECRET_ACCESS_KEY"),
		// 		os.Getenv("2OCWC53H91MGSASW5KFT"),
		// 		os.Getenv("wwDBKGIhOvlL9V5qLq4sR6eJjHVZCYFmi0jV6H1A"),
		// 		"",
		// 	)),
		// })
		// Инициализируем S3 Presign Клиент перед циклом сканирования строк СУБД
		// Используем официальные системные пакеты S3 SDK v2
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

		// bucketName := os.Getenv("S3_BUCKET_NAME")

		for _, s3Key := range s3Keys {
			s3Obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: &bucketName,
				Key:    aws.String(s3Key),
			})
			if err != nil {
				continue // Пропускаем поврежденный ключ, сохраняя поток
			}

			// Отрезаем вложенные папки для плоского имени файла внутри ZIP архива
			fileName := s3Key
			if idx := strings.LastIndex(s3Key, "/"); idx != -1 {
				fileName = s3Key[idx+1:]
			}

			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				s3Obj.Body.Close()
				continue
			}

			// io.Copy бинарно перекачивает Multipart-поток из S3 в ZIP HTTP-поток без выделения RAM
			_, _ = io.Copy(fileWriter, s3Obj.Body)
			s3Obj.Body.Close()
		}
	}
}
