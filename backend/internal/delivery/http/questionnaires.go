package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"         // ССЫЛКА
	"github.com/jackc/pgx/v5/pgxpool" // ССЫЛКА
)

// QuestionNode описывает структуру вопроса в присылаемом дереве анкеты
type QuestionNode struct {
	ID                    string         `json:"id"`      // Временный UUID фронтенда
	Section               string         `json:"section"` // general, sku
	SkuID                 *string        `json:"sku_id"`  // NULL для общих вопросов
	QuestionText          string         `json:"question_text"`
	ResponseType          string         `json:"response_type"` // text, number, photo, condition, select, date
	SortOrder             int            `json:"sort_order"`
	ConditionBranch       *bool          `json:"condition_branch"` // true (Да), false (Нет)
	ConditionOperator     *string        `json:"condition_operator"`
	ConditionValueText    []string       `json:"condition_value_text"`
	ConditionValueNumeric *float64       `json:"condition_value_numeric"`
	AvailableOptions      []string       `json:"available_options"`      // Варианты ответов
	AllowGallery          bool           `json:"allow_gallery"`          // Флаг доступа к галерее
	DependsOnQuestionID   interface{}    `json:"depends_on_question_id"` // Может быть string, float64, int или nil
	DependsOnOperator     *string        `json:"depends_on_operator"`
	DependsOnValue        *string        `json:"depends_on_value"`
	ChildQuestions        []QuestionNode `json:"child_questions"` // Рекурсивные подвопросы
}

// SaveQuestionnaireRequest описывает структуру запроса на создание новой версии анкеты
type SaveQuestionnaireRequest struct {
	ManufacturerID     int            `json:"manufacturer_id"`
	IsAutoCloneEnabled bool           `json:"is_auto_clone_enabled"` // <--- ДОБАВЛЕHО ДЛЯ ПРИЕМА ИЗ REACT
	Questions          []QuestionNode `json:"questions"`
}

// SaveQuestionnaireHandler создает новую версию анкеты и рекурсивно записывает дерево вопросов
func SaveQuestionnaireHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req SaveQuestionnaireRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Некорректный формат JSON пакета анкеты"}`))
			return
		}

		if req.ManufacturerID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Не указан корректный manufacturer_id"}`))
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		var currentMaxVersion int
		versionQuery := `SELECT COALESCE(MAX(version), 0) FROM questionnaires WHERE manufacturer_id = $1`
		err = tx.QueryRow(ctx, versionQuery, req.ManufacturerID).Scan(&currentMaxVersion)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		nextVersion := currentMaxVersion + 1

		var questionnaireID int
		insertQQuery := `
			INSERT INTO questionnaires (manufacturer_id, version, is_auto_clone_enabled) 
			VALUES ($1, $2, $3) RETURNING id`
		err = tx.QueryRow(ctx, insertQQuery, req.ManufacturerID, nextVersion, req.IsAutoCloneEnabled).Scan(&questionnaireID)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		uuidToDbIdMap := make(map[string]int)

		// Этап А: Сохраняем узлы и набиваем карту соответствия UUID -> SERIAL ID
		for _, qNode := range req.Questions {
			err = saveQuestionNode(ctx, tx, questionnaireID, nil, qNode, uuidToDbIdMap)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка при записи дерева вопросов анкеты: ` + err.Error() + `"}`))
				return
			}
		}

		// Этап Б: Проходим вторым каскадом и заменяем UUID на честные числовые ID PostgreSQL
		for _, qNode := range req.Questions {
			err = resolveInterQuestionDependencies(ctx, tx, qNode, uuidToDbIdMap)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Ошибка разрешения межвопросовых связей: ` + err.Error() + `"}`))
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "success", "version": ` + strconv.Itoa(nextVersion) + `}`))
	}
}

func saveQuestionNode(ctx context.Context, tx pgx.Tx, qID int, parentID *int, node QuestionNode, uuidMap map[string]int) error {

	node.QuestionText = strings.TrimSpace(node.QuestionText)
	if node.QuestionText == "" {
		return nil
	}

	insertQuery := `
		INSERT INTO questionnaire_questions (
			questionnaire_id, parent_question_id, section, sku_id, 
			question_text, response_type, sort_order, condition_branch, 
			condition_operator, condition_value_text, condition_value_numeric,
			available_options, allow_gallery,
			depends_on_question_id, depends_on_operator, depends_on_value
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`

	var newQuestionID int
	err := tx.QueryRow(ctx, insertQuery,
		qID,
		parentID,
		node.Section,
		node.SkuID,
		node.QuestionText,
		node.ResponseType,
		node.SortOrder,
		node.ConditionBranch,
		node.ConditionOperator,
		node.ConditionValueText,
		node.ConditionValueNumeric,
		node.AvailableOptions,
		node.AllowGallery,
		nil, // В Go передаем строго nil. Это запишет SQL NULL в ячейку depends_on_question_id
		node.DependsOnOperator,
		node.DependsOnValue,
	).Scan(&newQuestionID)

	if err != nil {
		return err
	}

	if node.ID != "" {
		uuidMap[node.ID] = newQuestionID
	}

	for _, childNode := range node.ChildQuestions {
		err = saveQuestionNode(ctx, tx, qID, &newQuestionID, childNode, uuidMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func resolveInterQuestionDependencies(ctx context.Context, tx pgx.Tx, node QuestionNode, uuidMap map[string]int) error {
	if node.DependsOnQuestionID != nil {
		var targetDbID int
		var found bool

		// БРОНЕБОЙНАЯ ПРОВЕРКА ТИПОВ: Защита от паники при nil, строках или числах
		switch v := node.DependsOnQuestionID.(type) {
		case string:
			if v != "" {
				targetDbID, found = uuidMap[v]
			}
		case float64:
			targetDbID = int(v)
			found = true
		case int:
			targetDbID = v
			found = true
		default:
			found = false
		}

		if found && targetDbID > 0 && node.ID != "" {
			currentDbID, exists := uuidMap[node.ID]
			if exists {
				updateQuery := `UPDATE questionnaire_questions SET depends_on_question_id = $1 WHERE id = $2`
				_, err := tx.Exec(ctx, updateQuery, targetDbID, currentDbID)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, childNode := range node.ChildQuestions {
		err := resolveInterQuestionDependencies(ctx, tx, childNode, uuidMap)
		if err != nil {
			return err
		}
	}

	return nil
}
