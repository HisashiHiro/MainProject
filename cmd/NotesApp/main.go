// @title Notes API
// @version 1.0
// @description API для управления заметками с JWT аутентификацией

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer {token}

package main

import (
	"MainProject/internal/handlers"
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "MainProject/cmd/NotesApp/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	goose "github.com/pressly/goose/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	pb "MainProject/api/proto"
	mygrpc "MainProject/internal/grpc"

	"google.golang.org/grpc"
)

// LoadEnv загружает переменные из .env
func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}
	return nil
}

func runMigrations() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	// Подключаемся к БД (для миграций)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Указываем директорию с миграциями
	if err := goose.Up(db, "./migrations"); err != nil {
		return err
	}
	log.Println("Migrations applied successfully")
	return nil
}

func main() {
	// Загрузка .env
	if err := LoadEnv(); err != nil {
		log.Fatalf("Ошибка загрузки .env: %v", err)
	}

	// Применяем миграции
	if err := runMigrations(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Проверяем обязательные переменные окружения
	requiredEnvVars := []string{"JWT_SECRET", "LOGIN", "PASSWORD"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Требуется установить переменную окружения: %s", envVar)
		}
	}

	// Создание контекста с возможностью отмены
	// cancel() будет вызван при получении сигнала ОС или по таймауту
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Гарантируем вызов cancel() при выходе из main()

	// Группа для отслеживания завершения горутин
	var wg sync.WaitGroup

	// Канал для перехвата системных сигналов (Ctrl+C, kill и т.п.)
	sigChan := make(chan os.Signal, 1)
	// Регистрация сигналов, которые нужно перехватывать
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создание репозиториЯ, передача контекста и WaitGroup
	repo := repository.NewRepository(ctx, &wg)
	if repo == nil {
		log.Fatalf("Не удалось инициализировать хранилище (PostgreSQL/Redis). Проверьте переменные окружения и доступность контейнеров.")
	}

	// Сервис заметок поверх репозитория
	noteService := service.NewNoteService(repo)

	// Создание и запуск gRPC сервера
	go func() {
		// Создание TCP listener
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Ошибка при создании listener: %v", err)
		}

		// Создание gRPC сервера
		s := grpc.NewServer()

		// Регистрация сервиса заметок
		notesServer := mygrpc.NewNotesServer(noteService)
		pb.RegisterNotesServiceServer(s, notesServer)

		log.Println("Запуск gRPC сервера на :50051")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Ошибка при запуске gRPC сервера: %v", err)
		}
	}()

	// Создание сервиса Планировщик, передача ему репозитория и WaitGroup
	scheduler := service.NewSchedulerService(repo, &wg)

	// Запуск планировщика в горутине
	wg.Add(1) // Регистрация горутины планировщика
	// Планировщик (каждые 5 сек генерирует сущности и отправляет их в канал репозитория)
	go func() {
		scheduler.Run(ctx, 5*time.Second)
		wg.Done() // Завершение
	}()

	// Gin-маршрутизация
	r := gin.Default()

	// Public routes
	r.POST("/login", handlers.HandleLogin)

	// Protected routes (требуют JWT)
	authorized := r.Group("/")
	authorized.Use(JWTMiddleware())
	{
		authorized.POST("/api/item", handlers.HandlePostItem(noteService))
		authorized.PUT("/api/item/:id", handlers.HandlePutItem(noteService))
		authorized.DELETE("/api/item/:id", handlers.HandleDeleteItem(noteService))
	}

	// Public GET
	r.GET("/api/item/:id", handlers.HandleGetItem(noteService))
	r.GET("/api/items", handlers.HandleGetItems(noteService))

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// HTTP‑сервер
	httpServer := &http.Server{
		Addr:    ":8080", // Порт 8080
		Handler: r,       // Маршрутизация запросов
	}

	// Запуск сервера в отдельной горутине
	go func() {
		log.Println("Запуск HTTP-сервера на :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Блокировка в режииме ожидания сигнала ОС (например, Ctrl+C)
	<-sigChan
	println("Получен сигнал завершения. Начинается плавное завершение приложения...")

	// Отмена контекста. Сигнал для всех горутин о необходимости завершиться
	cancel()

	// Ожидание, пока processEntities() обработает оставшиеся данные
	// 2 секунды на обработку буфера канала
	log.Println("Ожидание завершения обработки оставшихся данных...")
	time.Sleep(2 * time.Second)

	// Cохранение всех данных перед выходом
	log.Println("Сохранение данных перед завершением...")

	// Ожидание завершения всех горутин (до 10 сек)
	done := make(chan struct{})
	go func() {
		wg.Wait()   // Блокировка до тех пор, пока все горутины не вызовут wg.Done()
		close(done) // Сигнало о завершении всех горутин
	}()

	select {
	// Сигнал о завершении всех горутин (канал done закрыт)
	case <-done:
		println("Все горутины завершены успешно.")
	// Таймаут в 10 секунд (если какие-то горутины зависли)
	case <-time.After(10 * time.Second):
		println("Graceful shutdown timeout. Принудительное завершение.")
	}

	// После выхода из select main() завершается,
	// что приводит к остановке всего приложения

	// Пытаемся корректно остановить HTTP‑сервер
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при завершении сервера: %v", err)
	}

	// После остановки воркеров/сервера можно закрывать соединения к БД и Redis.
	repo.Close(ctx)

}

// JWTMiddleware — middleware для проверки JWT в заголовке Authorization
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
