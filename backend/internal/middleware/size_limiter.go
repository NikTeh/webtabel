package middleware

import (
	"net/http"
)

// MaxUploadSize устанавливает жесткий лимит в 20 МБ для XLSX и фото (Пункт 41 ТЗ)
const MaxUploadSize = 20 * 1024 * 1024

// RequestSizeLimiterMiddleware блокирует тяжелые пакеты до их попадания в память сервера
func RequestSizeLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Заворачиваем тело запроса в ограничитель MaxBytesReader
		r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

		// Предварительно проверяем заголовок Content-Length, если он указан клиентом
		if r.ContentLength > MaxUploadSize {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte(`{"error": "Превышен максимальный размер загружаемого файла (Лимит 20 МБ)"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
