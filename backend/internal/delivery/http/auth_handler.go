package http

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoginResponse описывает JSON-ответ, который ожидает мобильное приложение Pixel 4
type LoginResponse struct {
	Status string `json:"status"`
	Role   string `json:"role"`
	Name   string `json:"name"`
}

// MobileLoginHandler обрабатывает запрос авторизации от смартфона мерчандайзера
func MobileLoginHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		// Извлекаем пользователя, которого УЖЕ успешно проверил по базе пограничник BasicAuth
		user, ok := UserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Неверный логин или пароль"}`))
			return
		}

		// Формируем успешный ответ для Android-приложения
		resp := LoginResponse{
			Status: "success",
			Role:   user.Role,     // Вернет "merchandiser"
			Name:   user.FullName, // Вернет, например, "Николай Мерч"
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
