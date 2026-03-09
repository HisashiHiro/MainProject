package service

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"context"
	"fmt"
	"sync"
	"time"
)

// SchedulerService — сервис для периодической генерации и передачи сущностей в репозиторий
type SchedulerService struct {
	repo *repository.Repository // Ссылка на репозиторий для отправки данных

	// Указатель на WaitGroup для учёта горутин
	wg *sync.WaitGroup
}

// NewSchedulerService создаёт новый экземпляр сервиса планировщика
func NewSchedulerService(repo *repository.Repository, wg *sync.WaitGroup) *SchedulerService {
	return &SchedulerService{
		repo: repo,

		// Сохраняем указатель на WaitGroup
		wg: wg,
	}
}

// Run() периодически генерирует сущности и передаёт их в репозиторий
// Параметр interval определяет, как часто будут создаваться и отправляться новые сущности
// Все сущности упаковываются в срез и отправляются в канал репозитория
func (s *SchedulerService) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval) // Создаётся Ticker с заданным интервалом
	defer ticker.Stop()                // Гарантируем остановку тикера при завершении горутин

	for {
		// Используем select для одновременного ожидания:
		// 1. Срабатывания тикера (ticker.C)
		// 2. Сигнала отмены из контекста (ctx.Done())
		select {
		case <-ticker.C:
			// Генерация новой заметки
			note := model.NewNote("Пример", "Содержимое", []string{"тест"}, true)

			// Генерация нового пользователя с уникальным именем
			username := fmt.Sprintf("test-%d", time.Now().UnixNano())
			email := fmt.Sprintf("%s@example.com", username)
			user := model.NewUser(username, email, []byte("hash"))

			// Генерация нового тега с уникальным именем
			tagName := fmt.Sprintf("важное-%d", time.Now().UnixNano())
			tag := model.NewTag(tagName)

			// Упаковка всех сущностей в один срез
			entities := []model.Entity{note, user, tag}

			// Отправка среза сущностей в канал репозитория
			// Репозиторий самостоятельно обработает и распределит их по соответствующим слайсам
			s.repo.InputChannel() <- entities

		case <-ctx.Done():
			// Получен сигнал отмены (вызван cancel() в main)
			println("Scheduler завершает работу...")
			return // Выход из цикла и завершение горутины
		}
	}
}
