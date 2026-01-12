package main

import (
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Создание контекста с возможностью отмены
	// cancel() будет вызван при получении сигнала ОС или по таймауту
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Гарантируем вызов cancel() при выходе из main()

	// Канал для перехвата системных сигналов (Ctrl+C, kill и т.п.)
	sigChan := make(chan os.Signal, 1)
	// Регистрация сигналов, которые нужно перехватывать
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создание репозитория
	// Репозиторий автоматически запускает:
	// - горутину для обработки сущностей из канала (processEntities)
	// - горутину для логирования изменений (runLogger, интервал 200 мс)
	// - отслеживание команд завершения
	repo := repository.NewRepository(ctx)

	// Создание сервиса Планировщик, передавая ему репозиторий
	scheduler := service.NewSchedulerService(repo)

	// Запуск всех горутин
	// Планировщик (каждые 5 сек генерирует сущности и отправляет их в канал репозитория)
	go func() {
		scheduler.Run(ctx, 5*time.Second)
	}()

	// Блокировка в режииме ожидания сигнала ОС (например, Ctrl+C)
	<-sigChan
	println("Получен сигнал завершения. Начинается graceful shutdown...")

	// Отменяем контекст
	// Сигнал для всех горутин о необходимости завершиться
	cancel()

	// Основной поток продолжает работу
	// Все важные действия происходят в горутинах:
	// - планировщик отправляет данные
	// - репозиторий обрабатывает и логирует их
	select {
	// Время на завершение всех операций (10 секунд)
	case <-time.After(10 * time.Second):
		println("Graceful shutdown timeout. Принудительное завершение")
	case <-ctx.Done():
		// ctx.Done() закрывается после вызова cancel()
		println("Все горутины завершены успешно")
	}
}
