package http

import (
	// "context"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
)

// ImportMerchandisersHandler обрабатывает импорт XLSX файла мерчандайзеров и связывает их с СВ
func ImportMerchandisersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// 1. Техническая защита по ТЗ (Максимум 10 МБ)
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не удалось прочитать файл. Передайте его в поле 'file'"}`))
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

		// 2. Техническая защита: лимит на 2000 строк
		if len(rows) > 2000 {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte(`{"error": "Файл превышает лимит в 2000 строк"}`))
			return
		}

		if len(rows) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "В таблице отсутствуют данные для импорта"}`))
			return
		}

		// 3. Запуск транзакции для Full Overwrite механизма
		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось открыть транзакцию БД"}`))
			return
		}
		defer tx.Rollback(ctx)

		// Полностью очищаем старые связи СВ->МЕ и удаляем старых мерчандайзеров из таблицы users
		_, err = tx.Exec(ctx, "DELETE FROM supervisor_merchandisers")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка при очистке связей супервайзеров"}`))
			return
		}

		_, err = tx.Exec(ctx, "DELETE FROM users WHERE role = 'merchandiser'")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка при очистке таблицы мерчандайзеров"}`))
			return
		}

		insertedCount := 0

		// Проходим по строкам XLSX (пропуская заголовки на индексе 0)
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 6 {
				continue // Пропускаем неполные строки
			}

			// Парсим данные колонок
			idSvStr := strings.TrimSpace(row[0])
			idMeStr := strings.TrimSpace(row[2])
			fullNameMe := strings.TrimSpace(row[3])
			phoneMe := strings.TrimSpace(row[4])
			passwordMe := strings.TrimSpace(row[5])

			if idSvStr == "" || idMeStr == "" || phoneMe == "" || passwordMe == "" || fullNameMe == "" {
				continue // Пропускаем строки с пустыми ключевыми ячейками
			}

			extSvID, errSv := strconv.Atoi(idSvStr)
			extMeID, errMe := strconv.Atoi(idMeStr)
			if errSv != nil || errMe != nil {
				continue // Пропускаем строки с некорректными ID
			}

			// Проверяем, существует ли супервайзер с таким external_id в нашей базе данных
			var svInternalID int
			err = tx.QueryRow(ctx, "SELECT id FROM users WHERE external_id = $1 AND role = 'supervisor' LIMIT 1", extSvID).Scan(&svInternalID)
			if err != nil {
				// Если СВ не найден, мы не можем привязать к нему МЕ. Пропускаем строку ради целостности данных.
				continue
			}

			// Добавляем мерчандайзера в общую таблицу пользователей (роль 'merchandiser', пароль открытый по ТЗ)
			var meInternalID int
			insertMeQuery := `
				INSERT INTO users (external_id, phone, password, full_name, role, manufacturer_id)
				VALUES ($1, $2, $3, $4, 'merchandiser', NULL)
				RETURNING id`

			err = tx.QueryRow(ctx, insertMeQuery, extMeID, phoneMe, passwordMe, fullNameMe).Scan(&meInternalID)
			if err != nil {
				// Обработка дубликатов телефонов внутри файла импорта
				continue
			}

			// Связываем Супервайзера и Мерчандайзера в таблице связей
			insertLinkQuery := `
				INSERT INTO supervisor_merchandisers (supervisor_id, merchandiser_id)
				VALUES ($1, $2)`

			_, err = tx.Exec(ctx, insertLinkQuery, svInternalID, meInternalID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка связывания МЕ с Супервайзером в БД"}`))
				return
			}

			insertedCount++
		}

		// Фиксируем транзакцию
		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось зафиксировать изменения в PostgreSQL"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "imported_merchandisers_count": ` + strconv.Itoa(insertedCount) + `}`))
	}
}
