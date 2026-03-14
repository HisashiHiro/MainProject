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

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := goose.Up(db, "./migrations"); err != nil {
		return err
	}
	log.Println("Migrations applied successfully")
	return nil
}

func main() {
	// Загрузка .env
	if err := LoadEnv(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Ошибка загрузки .env: %v", err)
	}

	// Применяем миграции
	if err := runMigrations(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Проверяем обязательные переменные окружения
	requiredEnvVars := []string{"JWT_SECRET", "SESSION_SECRET"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Требуется установить переменную окружения: %s", envVar)
		}
	}

	// Создание контекста с возможностью отмены
	// cancel() будет вызван при получении сигнала ОС или по таймауту
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Гарантируем вызов cancel() при выходе из main()

	// Канал для перехвата системных сигналов (Ctrl+C, kill и т.п.)
	sigChan := make(chan os.Signal, 1)
	// Регистрация сигналов, которые нужно перехватывать
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ------- ВЫБОР ХРАНИЛИЩА ---------------------------
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "postgres" // значение по умолчанию
	}

	var repo interface{}
	switch dbType {
	case "postgres":
		repo = repository.NewPostgresRepository(ctx)
		if repo == nil {
			log.Fatalf("Не удалось инициализировать PostgreSQL")
		}
		defer repo.(*repository.PostgresRepository).Close(ctx)
	case "mongo":
		repo = repository.NewMongoRepository(ctx)
		if repo == nil {
			log.Fatalf("Не удалось инициализировать MongoDB")
		}
		defer repo.(*repository.MongoRepository).Close(ctx)
	default:
		log.Fatalf("Неизвестный тип БД: %s", dbType)
	}

	// Инициализация сервисов
	var noteRepo service.NoteRepository
	var userRepo service.UserRepository
	switch dbType {
	case "postgres":
		noteRepo = repo.(*repository.PostgresRepository)
		userRepo = repo.(*repository.PostgresRepository)
	case "mongo":
		noteRepo = repo.(*repository.MongoRepository)
		userRepo = repo.(*repository.MongoRepository)
	}

	noteService := service.NewNoteService(noteRepo) // Сервис заметок
	userService := service.NewUserService(userRepo) // Сервис пользователей

	// =========== gRPC сервер ===========
	grpcStop := make(chan struct{}) //Канал для отслеживания завершения сервера
	go func() {
		lis, err := net.Listen("tcp", ":50051") // Создание TCP listener
		if err != nil {
			log.Fatalf("Ошибка при создании listener: %v", err)
		}

		// Создание gRPC сервера
		s := grpc.NewServer()

		// Регистрация сервиса заметок
		notesServer := mygrpc.NewNotesServer(noteService)
		pb.RegisterNotesServiceServer(s, notesServer)
		log.Println("Запуск gRPC сервера на :50051")
		go func() {
			// Запуск внутренней горутины с s.Serve(), которая закрывает grpcStop при завершении
			if err := s.Serve(lis); err != nil {
				log.Printf("gRPC сервер остановлен: %v", err)
			}
			close(grpcStop)
		}()
		<-ctx.Done() // Ожидаем сигнала остановки
		log.Println("Остановка gRPC сервера...")
		s.GracefulStop() // Вызов корректного завершения активных RPC
	}()

	// ========== HTTP сервер (Gin) ==========
	// Gin-маршрутизация
	r := gin.Default()
	// Настройка сессий (используем cookie store)
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	r.Use(sessions.Sessions("notes-session", store))

	// Загружаем HTML-шаблоны из папки templates
	r.LoadHTMLGlob("templates/*")

	// Публичные веб-маршруты
	r.GET("/register", handlers.HandleWebRegisterPage)
	r.POST("/register", handlers.HandleWebRegister(userService))
	r.GET("/login", handlers.HandleWebLoginPage)
	r.POST("/login", handlers.HandleWebLogin(userService))
	r.GET("/logout", handlers.HandleWebLogout)

	// Защищённые веб-маршруты
	webAuth := r.Group("/")
	webAuth.Use(handlers.WebAuthMiddleware())
	{
		webAuth.GET("/", handlers.HandleWebIndex(noteService))
		webAuth.GET("/notes/new", handlers.HandleWebNewNote)
		webAuth.POST("/notes", handlers.HandleWebCreateNote(noteService))
		webAuth.GET("/notes/:id", handlers.HandleWebViewNote(noteService))
		webAuth.GET("/notes/:id/edit", handlers.HandleWebEditNote(noteService))
		webAuth.POST("/notes/:id", handlers.HandleWebUpdateNote(noteService))
		webAuth.POST("/notes/:id/delete", handlers.HandleWebDeleteNote(noteService))
	}

	// Публичные API
	apiPublic := r.Group("/api")
	{
		apiPublic.POST("/register", handlers.HandleRegister(userService))
		apiPublic.POST("/login", handlers.HandleLogin(userService))
		apiPublic.GET("/item/:id", handlers.HandleGetItem(noteService))
		apiPublic.GET("/items", handlers.HandleGetItems(noteService))
	}

	// Защищённые API (требуют JWT)
	apiAuth := r.Group("/api")
	apiAuth.Use(handlers.JWTMiddleware())
	{
		apiAuth.POST("/item", handlers.HandlePostItem(noteService))
		apiAuth.PUT("/item/:id", handlers.HandlePutItem(noteService))
		apiAuth.DELETE("/item/:id", handlers.HandleDeleteItem(noteService))
	}

	// Swagger документация
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// HTTP-сервер
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Запуск сервера в отдельной горутине
	httpStop := make(chan struct{}) //Канал для отслеживания завершения сервера
	go func() {
		log.Println("Запуск HTTP-сервера на :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP сервер остановлен: %v", err)
		}
		close(httpStop)
	}()

	// Блокировка в режииме ожидания сигнала ОС (например, Ctrl+C)
	<-sigChan
	println("Получен сигнал завершения. Начинается плавное завершение приложения...")
	cancel() // Отмена контекста. Сигнал для всех горутин о необходимости завершиться

	// Остановка HTTP сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при завершении HTTP-сервера: %v", err)
	}

	// Ожидание завершения серверов (максимум 10 секунд)
	select {
	case <-grpcStop:
		log.Println("gRPC сервер остановлен")
	case <-time.After(10 * time.Second):
		log.Println("Таймаут при остановке gRPC сервера")
	}

	select {
	case <-httpStop:
		log.Println("HTTP сервер остановлен")
	case <-time.After(10 * time.Second):
		log.Println("Таймаут при остановке HTTP сервера")
	}

	log.Println("Приложение завершено.")

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
