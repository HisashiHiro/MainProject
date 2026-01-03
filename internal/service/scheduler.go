package service

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"time"
)

// SchedulerService — сервис для периодической генерации и передачи сущностей в репозиторий
type SchedulerService struct {
	repo *repository.Repository // Ссылка на репозиторий для отправки данных
}

// NewSchedulerService создаёт новый экземпляр сервиса планировщика
func NewSchedulerService(repo *repository.Repository) *SchedulerService {
	return &SchedulerService{repo: repo}
}

// Run() периодически генерирует сущности и передаёт их в репозиторий
// Параметр interval определяет, как часто будут создаваться и отправляться новые сущности
// Все сущности упаковываются в срез и отправляются в канал репозитория
func (s *SchedulerService) Run(interval time.Duration) {
	ticker := time.NewTicker(interval) // Создаётся Ticker с заданным интервалом
	defer ticker.Stop()                // Гарантируем остановку тикера при завершении горутин

	for range ticker.C { // Бесконечный цикл по срабатываниям тикера
		// Генерация новой заметки
		note := model.NewNote("Пример", "Содержимое", []string{"тест"}, true)
		note.SetID(1) // Устанавливаем ID (в реальной системе должен быть уникальный генератор)

		// Генерация нового пользователя
		user := model.NewUser("test", "test@example.com", []byte("hash"))
		user.SetID(2)

		// Генерация новой сессии
		session := model.NewSession("session-123", 2, time.Now().Add(time.Hour), "127.0.0.1", "Chrome")

		// Генерация нового тега
		tag := model.NewTag("важное")
		tag.SetID(3)

		// Упаковка всех сущностей в один срез
		entities := []model.Entity{note, user, session, tag}

		// Отправка среза сущностей в канал репозитория
		// Репозиторий самостоятельно обработает и распределит их по соответствующим слайсам
		s.repo.InputChannel() <- entities
	}
}
