package main

import (
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"context"
	"log"
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
}
