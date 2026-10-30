package http

import (
	// "context"
	"fmt"
	// "log"
	"net/http"
	"os"

	// "path/filepath"
	// "strconv"
	// "time"

	"github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool" // ССЫЛКА
)

// func PhotoUploadHandler(db *pgxpool.Pool) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		ctx := r.Context()

// 		if r.Method != http.MethodPost {
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			return
// 		}

// 		answerIDStr := r.FormValue("answer_id")
// 		answerID, err := strconv.Atoi(answerIDStr)
// 		if err != nil || answerID <= 0 {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}

// 		// ИСПРАВЛЕНО ДЛЯ ПРОДАKШHА: Извлекаем честный полный путь, сгенерированный на смартфоне Николая
// 		s3Key := r.FormValue("photo_key")
// 		if s3Key == "" {
// 			w.WriteHeader(http.StatusBadRequest)
// 			w.Write([]byte(`{"error": "Не передан обязательный параметр photo_key"}`))
// 			return
// 		}

// 		file, _, err := r.FormFile("photo")
// 		if err != nil {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}
// 		defer file.Close()

// 		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
// 		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
// 		bucketName := os.Getenv("S3_BUCKET_NAME")

// 		if accessKey == "" || secretKey == "" || bucketName == "" {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}

// 		s3Client := s3.New(s3.Options{
// 			Region:       "ru-1",
// 			BaseEndpoint: aws.String("https://s3.twcstorage.ru"),
// 			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
// 				accessKey,
// 				secretKey,
// 				"",
// 			)),
// 		})

// 		// Загружаем бинарник в S3 строго по нашему оффлайн-ключу со слэшами
// 		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
// 			Bucket: &bucketName,
// 			Key:    &s3Key, // Сюда встанет "visits/22/имя.jpg"
// 			Body:   file,
// 		})
// 		if err != nil {
// 			log.Printf("🛑 СБОЙ S3 TIMEWEB: %v\n", err)
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}

// 		s3Url := fmt.Sprintf("https://s3.twcstorage.ru/%s/%s", bucketName, s3Key) //https://s3.twcstorage.ru

// 		insertQuery := `
// 			INSERT INTO report_photos (answer_id, s3_key, s3_url)
// 			VALUES ($1, $2, $3)
// 			ON CONFLICT DO NOTHING`

// 		_, err = db.Exec(ctx, insertQuery, answerID, s3Key, s3Url)
// 		if err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}

// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(`{"status": "success", "message": "Фото сохранено по иерархии папок", "s3_key": "` + s3Key + `"}`))
// 	}
// }

func PhotoUploadHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Параметр answer_id со смартфона мы игнорируем, так как он содержит смещенный ID вопроса
		s3Key := r.FormValue("photo_key") // Сюда прилетает точный "visits/137/gen_1793289412800.jpg"

		if s3Key == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Не передан photo_key"}`))
			return
		}

		file, _, err := r.FormFile("photo")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 1. Сначала загружаем бинарный файл в бакет S3 Timeweb Cloud
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		bucketName := os.Getenv("S3_BUCKET_NAME")

		// Вырезаем пустые дефолтные строки. Только жесткий отказ при утечке конфига
		if accessKey == "" || secretKey == "" || bucketName == "" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Конфигурация облачного хранилища S3 повреждена или отсутствует"}`))
			return
		}

		s3Client := s3.New(s3.Options{
			Region:       "ru-1",
			BaseEndpoint: aws.String("https://s3.twcstorage.ru"),
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
				accessKey, secretKey, "",
			)),
		})

		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &bucketName,
			Key:    &s3Key,
			Body:   file,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s3Url := fmt.Sprintf("https://s3.twcstorage.ru/%s/%s", bucketName, s3Key)

		// 2. Находим честный первичный ключ строки ra.id по точному совпадению s3Key
		var realAnswerRowID int
		findAnswerQuery := `
			SELECT id 
			FROM report_answers 
			WHERE value_text = $1 
			   OR value_text LIKE '%' || $1 || '%'
			LIMIT 1`

		err = db.QueryRow(ctx, findAnswerQuery, s3Key).Scan(&realAnswerRowID)
		if err != nil {
			// Если анкетный отчет по сети еще не успел дойти до PostgreSQL, прерываем транзакцию фото
			fmt.Printf("[S3] Строка ответа еще не создана в БД для ключа: %s (%v)\n", s3Key, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 3. Записываем связь в таблицу индекса report_photos без конфликтов внешних ключей
		insertQuery := `
			INSERT INTO report_photos (answer_id, s3_key, s3_url)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`

		_, err = db.Exec(ctx, insertQuery, realAnswerRowID, s3Key, s3Url)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// fmt.Printf("🚀 [БД УСПЕХ] Фото привязано к ra.id = %d, Маска S3: %s\n", realAnswerRowID, s3Key)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "s3_key": "` + s3Key + `"}`))
	}
}
