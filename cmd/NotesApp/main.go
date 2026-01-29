package main

import (
	"MainProject/internal/handlers"
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
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
	// Создание сервиса Планировщик, передача ему репозитория и WaitGroup
	scheduler := service.NewSchedulerService(repo, &wg)

	// Запуск планировщика в горутине
	wg.Add(1) // Регистрация горутины планировщика
	// Планировщик (каждые 5 сек генерирует сущности и отправляет их в канал репозитория)
	go func() {
		scheduler.Run(ctx, 5*time.Second)
		wg.Done() // Завершение
	}()

	// HTTP‑сервер
	httpServer := &http.Server{
		Addr:    ":8080",           // Порт 8080
		Handler: setupRoutes(repo), // Маршрутизация запросов
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
	repo.Flush()

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

}

// setupRoutes — настраивает HTTP‑маршрутизацию
// Принимает репозиторий, чтобы обработчики могли работать с данными
// Возвращает http.Handler (мультиплексор маршрутов)
func setupRoutes(repo *repository.Repository) http.Handler {
	mux := http.NewServeMux() // Создаём маршрутизатор

	// Маршрут для создания заметки (POST /api/item)
	mux.HandleFunc("/api/item", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.HandlePostItem(w, r, repo) // Обработчик создания
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	})

	// Маршрут для операций с конкретной заметкой (-GET/PUT/DELETE /api/item/{id})
	mux.HandleFunc("/api/item/", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Path[len("/api/item/"):]
		if idStr == "" {
			http.Error(w, "ID не указан", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handlers.HandleGetItem(w, r, repo, idStr)
		case http.MethodPut:
			handlers.HandlePutItem(w, r, repo, idStr)
		case http.MethodDelete:
			handlers.HandleDeleteItem(w, r, repo, idStr)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	})

	// Маршрут для получения списка всех заметок (GET /api/items)
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}
		handlers.HandleGetItems(w, r, repo)
	})

	return mux
}
