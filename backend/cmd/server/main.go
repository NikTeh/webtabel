package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	deliveryHttp "mg-backend/internal/delivery/http"
	"mg-backend/internal/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Встроенная функция для автоматического чтения файла .env и записи параметров в систему
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// Если файла нет, сервер просто продолжит работу на стандартных настройках системы
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Пропускаем пустые строки и комментарии со знаком #
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Разделяем строку на имя параметра и его значение по первому знаку равенства
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}

func main() {
	// Автоматически загружаем параметры из текстового файла при старте бэкенда
	loadEnv()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА ДЕПЛОЯ: Переменная DATABASE_URL не задана в файле .env\n")
	}

	// Проверка наличия ключей S3 перед запуском
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("S3_BUCKET_NAME") == "" {
		log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА ДЕПЛОЯ: Ключи доступа S3 Timeweb Cloud не обнаружены в .env\n")
	}

	// ИСПРАВЛЕHО ДЛЯ ПРОДАКШHА: Контекст для безопасной проверки связи при старте сервера
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()

	fmt.Printf("Подключение к PostgreSQL...\n")
	// Пул создается на независимом фоне context.Background(), чтобы не обрывать долгие Multipart-запросы
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Не удалось создать пул соединений: %v\n", err)
	}
	defer dbPool.Close()

	// Выполняем проверочный пинг с коротким таймаутом
	err = dbPool.Ping(pingCtx)
	if err != nil {
		log.Fatalf("База данных недоступна: %v\n", err)
	} else {
		fmt.Printf("Успешное подключение к PostgreSQL!\n")
	}

	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Разрешаем запросы с любого источника (включая ваш React веб-кабинет)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Если прилетел предварительный проверочный запрос браузера OPTIONS — мгновенно отдаем 200 OK
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	// Регистрация всех существующих маршрутов (окружающий исходный контекст)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))

	// 2. ЗАЩИЩЕННЫЙ ЭНДПОИНТ (Доступен ВСЕМ авторизованным пользователям)
	// ИСПРАВЛЕНО: Объявляем функцию отдельно через тип http.HandlerFunc
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := deliveryHttp.UserFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok", "message":"Доступ получен", "user_id":%d, "name":"%s", "role":"%s"}`,
			user.ID, user.FullName, user.Role)
	})

	mux.Handle("/api/v1/test/protected", deliveryHttp.BasicAuthMiddleware(dbPool)(protectedHandler))

	// 3. ЗАЩИЩЕННЫЙ ЭНДПОИНТ ТОЛЬКО ДЛЯ ВЛАДЕЛЬЦА (owner)
	ownerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok", "message":"Приветствую Админ! Доступ в ЛК разрешен."}`))
	})
	mux.Handle("/api/v1/test/owner", deliveryHttp.BasicAuthMiddleware(dbPool)(deliveryHttp.RequireRoleMiddleware("owner")(ownerHandler)))

	// ====================================================================
	// МАРШРУТЫ АДМИНИСТРАТИВНОЙ ПАНЕЛИ ВЛАДЕЛЬЦА (owner)
	// ====================================================================

	// 4. Импорт пользователей из XLSX
	mux.Handle("/api/v1/owner/users/import",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ImportUsersHandler(dbPool),
			),
		),
	)

	// 5. Ручное управление пользователями
	mux.Handle("/api/v1/owner/users",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ManageUsersHandler(dbPool),
			),
		),
	)

	// 6. Импорт мерчандайзеров из XLSX
	mux.Handle("/api/v1/owner/merchandisers/import",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ImportMerchandisersHandler(dbPool),
			),
		),
	)

	// 7. Импорт торговых точек из XLSX
	mux.Handle("/api/v1/owner/stores/import",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ImportStoresHandler(dbPool),
			),
		),
	)

	// 8. Ручное управление карточками Производителей
	mux.Handle("/api/v1/owner/manufacturers",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ManageManufacturersHandler(dbPool),
			),
		),
	)

	// 9. Импорт SKU-листа Производителя из XLSX
	mux.Handle("/api/v1/owner/manufacturers/skus/import",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ImportSkusHandler(dbPool),
			),
		),
	)

	// 10. Импорт списка привязанных точек Производителя из XLSX
	mux.Handle("/api/v1/owner/manufacturers/stores/import",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.ImportManufacturerStoresHandler(dbPool),
			),
		),
	)

	// 11. Получение матрицы дистрибьюции Производителя
	mux.Handle("/api/v1/owner/matrices",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.GetMatrixHandler(dbPool),
			),
		),
	)

	// 12. Сохранение матрицы дистрибьюции Производителя
	mux.Handle("/api/v1/owner/matrices/save",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.SaveMatrixHandler(dbPool),
			),
		),
	)

	// 12-Б. Сохранение новой версии анкеты Производителя целиком
	mux.Handle("/api/v1/owner/questionnaires/save",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.SaveQuestionnaireHandler(dbPool),
			),
		),
	)

	// 12-В. Скачивание структуры текущей активной анкеты для ЛК Владельца
	mux.Handle("/api/v1/owner/questionnaires/current",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.GetCurrentQuestionnaireHandler(dbPool),
			),
		),
	)

	// 12-Г. Точечное косметическое обновление формулировок вопросов без повышения версии
	mux.Handle("/api/v1/owner/questionnaires/update-text",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("owner")(
				deliveryHttp.UpdateQuestionTextsHandler(dbPool),
			),
		),
	)

	// ====================================================================
	// МАРШРУТЫ МОБИЛЬНОГО ПРИЛОЖЕНИЯ МЕРЧАНДАЙЗЕРА (merchandiser)
	// ====================================================================

	// Эндпоинт входа для мобильного приложения Pixel 4
	mux.Handle("/api/v1/auth/login",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.MobileLoginHandler(dbPool),
		),
	)

	// Эндпоинт утренней синхронизации для мерчандайзеров
	mux.Handle("/api/v1/me/sync",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("merchandiser")(
				deliveryHttp.MerchandiserSyncHandler(dbPool),
			),
		),
	)

	// 13. Старт визита мерчандайзера
	mux.Handle("/api/v1/me/visits/start",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("merchandiser")(
				deliveryHttp.StartVisitHandler(dbPool),
			),
		),
	)

	// 14. Завершение визита мерчандайзера
	mux.Handle("/api/v1/me/visits/finish",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("merchandiser")(
				deliveryHttp.FinishVisitHandler(dbPool),
			),
		),
	)

	// 15. Пакетная синхронизация отчётов и ответов на анкеты с Android
	mux.Handle("/api/v1/me/visits/reports/sync",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("merchandiser")(
				deliveryHttp.SyncVisitReportsHandler(dbPool),
			),
		),
	)

	// 16. Загрузка полевых фотографий выкладки товаров в S3-облако Яндекс Облака
	mux.Handle("/api/v1/me/photos/upload",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("merchandiser")(
				deliveryHttp.PhotoUploadHandler(dbPool),
			),
		),
	)

	// ====================================================================
	// МАРШРУТЫ ВЕБ-КАБИНЕТА СУПЕРВАЙЗЕРА (supervisor)
	// ====================================================================

	// 17. Получение списка подотчетных торговых точек текущего Супервайзера
	mux.Handle("/api/v1/sv/stores",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("supervisor")(
				deliveryHttp.GetSvStoresHandler(dbPool),
			),
		),
	)

	// 18. Назначение, замена и удаление МЕ с торговой точки Супервайзером
	mux.Handle("/api/v1/sv/stores/assign",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("supervisor")(
				deliveryHttp.AssignMerchandiserToStoreHandler(dbPool),
			),
		),
	)

	// 19. Построчное ручное управление своей локальной командой МЕ
	mux.Handle("/api/v1/sv/merchandisers",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.RequireRoleMiddleware("supervisor")(
				deliveryHttp.ManageSvMerchandisersHandler(dbPool),
			),
		),
	)

	// ====================================================================
	// ДОБАВЛЕНО: ЭНДПОИНТЫ ГЕНЕРАЦИИ СТРЫМИНГОВЫХ ВЫГРУЗОК XLSX ОТЧЕТОВ
	// ====================================================================
	mux.Handle("/api/v1/reports/visits/export",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.ExportVisitsXlsxHandler(dbPool),
		),
	)

	mux.Handle("/api/v1/reports/data/export",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.ExportAnswersXlsxHandler(dbPool),
		),
	)

	mux.Handle("/api/v1/reports/photos/export",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.ExportPhotosZipHandler(dbPool),
		),
	)

	// ====================================================================
	// ДОБАВЛЕНО: ЭНДПОИНТЫ ЖИВЫХ JSON-ДАННЫХ ДЛЯ ОТРИСОВКИ ТАБЛИЦ В ЛК
	// ====================================================================
	mux.Handle("/api/v1/reports/visits",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.GetVisitsReportHandler(dbPool),
		),
	)

	mux.Handle("/api/v1/reports/photos",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.GetPhotosReportHandler(dbPool),
		),
	)

	mux.Handle("/api/v1/reports/data",
		deliveryHttp.BasicAuthMiddleware(dbPool)(
			deliveryHttp.GetAnswersReportHandler(dbPool),
		),
	)

	mux.Handle("/api/v1/reports/filters/regions", deliveryHttp.BasicAuthMiddleware(dbPool)(deliveryHttp.GetReportRegionsHandler(dbPool)))
	mux.Handle("/api/v1/reports/filters/supervisors", deliveryHttp.BasicAuthMiddleware(dbPool)(deliveryHttp.GetReportSupervisorsHandler(dbPool)))
	mux.Handle("/api/v1/config/map", deliveryHttp.BasicAuthMiddleware(dbPool)(deliveryHttp.GetMapConfigHandler(dbPool)))
	mux.Handle("/api/v1/reports/filters/merchandisers", deliveryHttp.BasicAuthMiddleware(dbPool)(deliveryHttp.GetReportMerchandisersHandler(dbPool)))

	// Оборачиваем глобальный роутер Go в лимитер размера
	// finalHandler := middleware.RequestSizeLimiterMiddleware(http.DefaultServeMux)

	// fmt.Printf("=== Старт промышленного бэкенда MG ===\n")
	// fmt.Printf("Сервер запущен на порту: %s\n", port)

	// if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
	// 	log.Fatalf("Критическая ошибка запуска сервера: %v\n", err)
	// }

	finalHandler := middleware.RequestSizeLimiterMiddleware(corsHandler(mux))

	fmt.Printf("=== Старт промышленного защищенного бэкенда MG ===\n")
	if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
		log.Fatalf("Критическая ошибка запуска сервера: %v\n", err)
	}
}
