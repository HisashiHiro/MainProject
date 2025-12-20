package main

import (
	"MainProject/internal/repository"
	"MainProject/internal/service"
	"time"
)

func main() {
	// Создаём репозиторий
	repo := repository.NewRepository()

	// Создаём сервис планировщика
	scheduler := service.NewSchedulerService(repo)

	// Запускаем с интервалом 5 секунд
	go scheduler.Run(5 * time.Second)

	// Основной поток продолжает работу
	select {}
}
