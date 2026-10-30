package http

import (
	"encoding/json"
	// "fmt"
	"net/http"
	"strconv"

	// "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnswerData описывает один конкретный ответ на вопрос анкеты
type AnswerData struct {
	QuestionID  int      `json:"question_id"`
	SkuID       *string  `json:"sku_id"` // NULL для общих вопросов
	ValueText   *string  `json:"value_text"`
	ValueNumber *float64 `json:"value_number"`
	// ValueBoolean *bool    `json:"value_boolean"`
}

// ManufacturerReportData описывает отчёт по конкретному производителю внутри пакета
type ManufacturerReportData struct {
	ManufacturerID  int          `json:"manufacturer_id"`
	QuestionnaireID int          `json:"questionnaire_id"` // Версия анкеты, скачанная утром при синхронизации
	Answers         []AnswerData `json:"answers"`
}

// SyncReportsRequest описывает структуру входящего JSON-пакета для синхронизации отчётов визита
type SyncReportsRequest struct {
	VisitID string                   `json:"visit_id"` // UUID визита
	Reports []ManufacturerReportData `json:"reports"`
}

// SyncVisitReportsHandler принимает с Android-устройства результаты заполнения анкет
func SyncVisitReportsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Метод не поддерживается"}`))
			return
		}

		var req SyncReportsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		// ПРОМЫШЛЕHHЫЙ СТАHДАРТ: Пустой пакет отчетов легален (Кейс: только выкладка). Блокируем только пустой UUID.
		if req.VisitID == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Передан пустой идентификатор визита visit_id"}`))
			return
		}

		// Если Николай сдал пустой визит (0 ответов), мы сразу возвращаем HTTP 200 OK, позволяя цепочке коммитов завершиться в PostgreSQL
		if len(req.Reports) == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "synced_reports_count": 0, "synced_answers_count": 0, "message": "Пустой отчет выкладки принят бэкендом"}`))
			return
		}

		// Открываем транзакцию для обеспечения атомарности записи пакета данных
		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось открыть транзакцию БД"}`))
			return
		}
		defer tx.Rollback(ctx)

		// Проверяем, существует ли вообще такой визит в системе
		var visitExists bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM visits WHERE id = $1)", req.VisitID).Scan(&visitExists)
		if err != nil || !visitExists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Визит с указанным UUID не найден в базе данных"}`))
			return
		}

		reportsSynced := 0
		answersSynced := 0

		// Обрабатываем каждый отчёт по производителю в пакете
		for _, reportData := range req.Reports {
			if reportData.ManufacturerID <= 0 || reportData.QuestionnaireID <= 0 {
				continue
			}

			// Реализация Full Overwrite контура для конкретного производителя в рамках этого визита.
			// Сначала удаляем старый отчёт (каскадно удалятся и старые ответы из report_answers).
			// Это гарантирует отсутствие дублирования при повторной отправке данных из оффлайн-очереди Android.
			_, err = tx.Exec(ctx, `
				DELETE FROM visit_manufacturer_reports 
				WHERE visit_id = $1 AND manufacturer_id = $2`,
				req.VisitID, reportData.ManufacturerID,
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка очистки предыдущих отчётов производителя"}`))
				return
			}

			// Создаем новую запись отчёта производителя с жёсткой фиксацией исторической версии анкеты
			var reportInternalID int
			insertReportQuery := `
				INSERT INTO visit_manufacturer_reports (visit_id, manufacturer_id, questionnaire_id, started_at, completed_at)
				VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`

			err = tx.QueryRow(ctx, insertReportQuery, req.VisitID, reportData.ManufacturerID, reportData.QuestionnaireID).Scan(&reportInternalID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка сохранения заголовка отчёта производителя"}`))
				return
			}

			for _, ans := range reportData.Answers {

				// ИСПРАВЛЕНО: Бронебойная защита внешнего ключа СУБД.
				// Теперь мы ловим и слово "DEFAULT" / "default", принудительно сбрасывая его в nil (SQL NULL),
				// чтобы база данных PostgreSQL не ругалась на отсутствие товара в таблице skus!
				var finalSkuID *string = ans.SkuID
				if ans.SkuID != nil {
					trimmedSku := *ans.SkuID
					if trimmedSku == "" || trimmedSku == "general" || trimmedSku == "General" || trimmedSku == "DEFAULT" || trimmedSku == "default" {
						finalSkuID = nil
					}
				}

				insertAnswerQuery := `
					INSERT INTO report_answers (report_id, question_id, sku_id, value_text, value_number)
					VALUES ($1, $2, $3, $4, $5)`

				// Используем очищенный безопасный finalSkuID
				_, err = tx.Exec(ctx, insertAnswerQuery,
					reportInternalID,
					ans.QuestionID,
					finalSkuID,
					ans.ValueText,
					ans.ValueNumber,
				)

				if err != nil {
					// fmt.Println("🛑 СКРЫТАЯ ОШИБКА В ЦИКЛЕ INSERT:", err)
					continue
				}
				answersSynced++
			}

			reportsSynced++
		}

		// Фиксируем транзакцию в PostgreSQL
		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Не удалось зафиксировать транзакцию отчётов"}`))

			// ВЫВОДИМ РЕАЛЬНУЮ ПРИЧИНУ СБОЯ ТРАНЗАКЦИИ ОТ СУБД
			// fmt.Println("КРИТИЧЕСКАЯ ОШИБКА COMMIT:", err)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "synced_reports_count": ` + strconv.Itoa(reportsSynced) + `, "synced_answers_count": ` + strconv.Itoa(answersSynced) + `}`))
	}
}
