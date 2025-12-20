package service

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"time"
)

// SchedulerService — сервис для периодической генерации и передачи сущностей.
type SchedulerService struct {
	repo *repository.Repository
}

// NewSchedulerService создаёт новый сервис.
func NewSchedulerService(repo *repository.Repository) *SchedulerService {
	return &SchedulerService{repo: repo}
}

// Run периодическ генерирует сущности и передаёт их в репозиторий.
func (s *SchedulerService) Run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		note := model.NewNote("Пример", "Содержимое", []string{"тест"}, true)
		note.SetID(1)

		user := model.NewUser("test", "test@example.com", []byte("hash"))
		user.SetID(2)

		session := model.NewSession("session-123", 2, time.Now().Add(time.Hour), "127.0.0.1", "Chrome")

		tag := model.NewTag("важное")
		tag.SetID(3)

		entities := []model.Entity{note, user, session, tag}
		s.repo.SaveEntities(entities) // Передача сущностей в репозиторий
	}
}
