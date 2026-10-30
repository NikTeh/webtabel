package http

import (
	// "context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MobileQuestionResponse повторно используется или расширяется для ЛК
type CurrentAnketaQuestion struct {
	ID                    int      `json:"id"`
	QuestionnaireID       int      `json:"questionnaire_id"`
	Section               string   `json:"section"`
	ParentQuestionID      *int     `json:"parent_question_id"`
	QuestionText          string   `json:"question_text"`
	ResponseType          string   `json:"response_type"`
	AvailableOptions      []string `json:"available_options"`
	SortOrder             int      `json:"sort_order"`
	ConditionBranch       *bool    `json:"condition_branch"`
	ConditionOperator     *string  `json:"condition_operator"`
	ConditionValueText    []string `json:"condition_value_text"`
	ConditionValueNumeric *float64 `json:"condition_value_numeric"`
	AllowGallery          bool     `json:"allow_gallery"`
	DependsOnQuestionID   *int     `json:"depends_on_question_id"` // Ссылка на внешний вопрос
	DependsOnOperator     *string  `json:"depends_on_operator"`
	DependsOnValue        *string  `json:"depends_on_value"` // Проверочный текст промо
}

// GetCurrentQuestionnaireHandler скачивает структуру активной анкеты для ЛК Владельца
func GetCurrentQuestionnaireHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		mIDStr := r.URL.Query().Get("manufacturer_id")
		mID, err := strconv.Atoi(mIDStr)
		if err != nil || mID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Параметр manufacturer_id указан некорректно"}`))
			return
		}

		// 1. Находим ID последней (максимальной) версии анкеты производителя
		var qID int
		var version int
		var isAutoClone bool

		err = db.QueryRow(ctx, `
			SELECT id, version, is_auto_clone_enabled 
			FROM questionnaires 
			WHERE manufacturer_id = $1 
			ORDER BY version DESC LIMIT 1`, mID).Scan(&qID, &version, &isAutoClone)

		if err != nil {
			if err == pgx.ErrNoRows {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"version": 0, "is_auto_clone_enabled": false, "questions": []}`))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 2. Вытягиваем все вопросы этой версии, включая новые аналитические поля зависимостей
		query := `
			SELECT id, questionnaire_id, section, parent_question_id, question_text, response_type,
			       available_options, sort_order, condition_branch, condition_operator,
			       condition_value_text, condition_value_numeric, allow_gallery,
			       depends_on_question_id, depends_on_operator, depends_on_value
			FROM questionnaire_questions
			WHERE questionnaire_id = $1
			ORDER BY sort_order ASC`

		rows, err := db.Query(ctx, query, qID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		questions := []CurrentAnketaQuestion{}
		for rows.Next() {
			var q CurrentAnketaQuestion
			err := rows.Scan(
				&q.ID, &q.QuestionnaireID, &q.Section, &q.ParentQuestionID, &q.QuestionText, &q.ResponseType,
				&q.AvailableOptions, &q.SortOrder, &q.ConditionBranch, &q.ConditionOperator,
				&q.ConditionValueText, &q.ConditionValueNumeric, &q.AllowGallery,
				&q.DependsOnQuestionID, &q.DependsOnOperator, &q.DependsOnValue,
			)
			if err != nil {
				continue
			}
			if q.AvailableOptions == nil {
				q.AvailableOptions = []string{}
			}
			if q.ConditionValueText == nil {
				q.ConditionValueText = []string{}
			}
			questions = append(questions, q)
		}

		// Упаковываем версию и плоский массив вопросов со связями parent/depends наружу
		resp := map[string]interface{}{
			"questionnaire_id":      qID,
			"version":               version,
			"is_auto_clone_enabled": isAutoClone, // <--- Отдаем статус на фронтенд React
			"questions":             questions,
		}
		json.NewEncoder(w).Encode(resp)

	}
}

// TextUpdateRequest описывает одну строку изменения формулировки
type TextUpdateItem struct {
	ID   int    `json:"id"`
	Text string `json:"question_text"`
}

// UpdateQuestionTextsHandler точечно правит тексты вопросов без генерации новой версии анкеты
func UpdateQuestionTextsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var items []TextUpdateItem
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON"}`))
			return
		}

		if len(items) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Массив изменений пуст"}`))
			return
		}

		// Открываем изолированную транзакцию для защиты от частичного обновления при сбоях сети
		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		updatedCount := 0
		for _, item := range items {
			item.Text = strings.TrimSpace(item.Text)
			if item.ID <= 0 || item.Text == "" {
				continue
			}

			// Точечный UPDATE строго по системному первичному ключу id questionnaire_questions
			query := `UPDATE questionnaire_questions SET question_text = $1 WHERE id = $2`
			cmdTag, err := tx.Exec(ctx, query, item.Text, item.ID)
			if err == nil {
				updatedCount += int(cmdTag.RowsAffected())
			}
		}

		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Ошибка фиксации изменений"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "Формулировки успешно обновлены", "updated_count": ` + strconv.Itoa(updatedCount) + `}`))
	}
}
