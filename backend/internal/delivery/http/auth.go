package http

import (
	"context"
	// "fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User - структура авторизованного пользователя для всей системы MG
type User struct {
	ID             int
	ExternalID     *int // Используем указатель, так как в БД поле nullable
	Phone          string
	FullName       string
	Role           string // owner, supervisor, merchandiser, manufacturer
	ManufacturerID *int   // Заполнен только для роли manufacturer
}

// Ключ для безопасного хранения пользователя в контексте запроса
type userContextKey struct{}

// BasicAuthMiddleware перехватывает запрос, проверяет "телефон+пароль" и сохраняет пользователя в контексте
func BasicAuthMiddleware(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// 1. Извлекаем данные Basic Auth
			phone, password, ok := r.BasicAuth()
			if !ok || phone == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="MG API"`)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Требуется авторизация Basic Auth"}`))
				return
			}

			phone = strings.TrimSpace(phone)

			// 2. Ищем пользователя в БД, используя контекст текущего HTTP-запроса r.Context()
			var user User
			var storedPassword string

			query := `
				SELECT id, external_id, phone, password, full_name, role, manufacturer_id 
				FROM users 
				WHERE phone = $1 LIMIT 1`

			err := db.QueryRow(r.Context(), query, phone).Scan(
				&user.ID,
				&user.ExternalID,
				&user.Phone,
				&storedPassword,
				&user.FullName,
				&user.Role,
				&user.ManufacturerID,
			)

			if err != nil || storedPassword != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="MG API"`)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Неверный телефон или пароль"}`))
				return
			}

			// 3. Кладем структуру пользователя в контекст запроса
			ctx := context.WithValue(r.Context(), userContextKey{}, &user)

			// Передаем запрос дальше с обновленным контекстом
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoleMiddleware проверяет, соответствует ли роль пользователя из контекста разрешенным ролям
func RequireRoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// Достаем пользователя из контекста
			user, ok := UserFromContext(r.Context())
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Пользователь не авторизован"}`))
				return
			}

			// Проверяем роль
			roleAllowed := false
			for _, allowedRole := range allowedRoles {
				if user.Role == allowedRole {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Доступ запрещен для вашей роли"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext - сквозная функция, которую будет вызывать любой модуль (отчеты, точки), чтобы узнать кто делает запрос
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*User)
	return user, ok
}
