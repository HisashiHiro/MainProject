package main

import (
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"time"
)

func main() {
	// Создание репозитория
	// Репозиторий автоматически запускает:
	// - горутину для обработки сущностей из канала (processEntities)
	// - горутину для логирования изменений (runLogger, интервал 200 мс)
	repo := repository.NewRepository()

	// Создание сервиса Планировщик, передавая ему репозиторий
	scheduler := service.NewSchedulerService(repo)

	// Запуск всех горутин
	// Планировщик (каждые 5 сек генерирует сущности и отправляет их в канал репозитория)
	go scheduler.Run(5 * time.Second)

	// Основной поток продолжает работу
	// Все важные действия происходят в горутинах:
	// - планировщик отправляет данные
	// - репозиторий обрабатывает и логирует их
	select {}
}
