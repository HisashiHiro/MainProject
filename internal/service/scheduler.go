package service

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"fmt"
	"time"
)

// SchedulerService — сервис для периодической генерации и передачи сущностей в репозиторий
type SchedulerService struct {
	repo *repository.Repository // Ссылка на репозиторий для отправки данных

	// Поля для логирования изменений
	// Предыдущие размеры слайсов (для отслеживания изменений)
	prevNotes    int
	prevUsers    int
	prevSessions int
	prevTags     int
}

// NewSchedulerService создаёт новый экземпляр сервиса планировщика
func NewSchedulerService(repo *repository.Repository) *SchedulerService {
	return &SchedulerService{
		repo: repo,

		// Инициализация предыдущих размеров слайсов нулями (начальное состояние)
		prevNotes:    0,
		prevUsers:    0,
		prevSessions: 0,
		prevTags:     0,
	}
}

// Run() периодически генерирует сущности и передаёт их в репозиторий
// Параметр interval определяет, как часто будут создаваться и отправляться новые сущности
// Все сущности упаковываются в срез и отправляются в канал репозитория
func (s *SchedulerService) Run(interval time.Duration) {

	// Запускаем горутину для логирования изменений (каждые 200 мс)
	go s.runLogger(200 * time.Millisecond)

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

// runLogger — горутина, которая периодически проверяет изменения в слайсах и логирует новые элементы.
// Параметр interval определяет, как часто проверять изменения
// Использует Ticker для периодического вызова метода logChanges()
func (s *SchedulerService) runLogger(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.logChanges()
	}
}

// logChanges выполняет:
// 1. Получение текущих данных из репозитория (с мьютексами)
// 2. Сравнение текущих размеров слайсов с предыдущими значениями
// 3. Логирование только новых элементов (которые появились с последнего вызова)
// 4. Обновление предыдущих размеров для следующего цикла
func (s *SchedulerService) logChanges() {
	// Получение текущих данных из репозитория (методы уже содержат мьютексы)
	notes := s.repo.GetNotes()
	users := s.repo.GetUsers()
	sessions := s.repo.GetSessions()
	tags := s.repo.GetTags()

	// Текущие размеры слайсов
	currentNotes := len(notes)
	currentUsers := len(users)
	currentSessions := len(sessions)
	currentTags := len(tags)

	// Проверка и логирование новых заметок
	// Проверка изменения размеров слайсов с последнего раза
	if currentNotes > s.prevNotes {
		added := currentNotes - s.prevNotes
		fmt.Printf("[LOG] Добавлено заметок: %d\n", added)
		for i := s.prevNotes; i < currentNotes; i++ {
			fmt.Printf("  Note: ID=%v, Title=%s\n", notes[i].ID(), notes[i].(*model.Note).Title())
		}
	}

	// Проверка и логирование новых пользователей
	if currentUsers > s.prevUsers {
		added := currentUsers - s.prevUsers
		fmt.Printf("[LOG] Добавлено пользователей: %d\n", added)
		for i := s.prevUsers; i < currentUsers; i++ {
			fmt.Printf("  User: ID=%v, Username=%s\n", users[i].ID(), users[i].(*model.User).Username())
		}
	}

	// Проверка и логирование новых сессий
	if currentSessions > s.prevSessions {
		added := currentSessions - s.prevSessions
		fmt.Printf("[LOG] Добавлено сессий: %d\n", added)
		for i := s.prevSessions; i < currentSessions; i++ {
			fmt.Printf("  Session: ID=%v, UserID=%d\n", sessions[i].ID(), sessions[i].(*model.Session).UserID())
		}
	}

	// Проверка и логирование новых тегов
	if currentTags > s.prevTags {
		added := currentTags - s.prevTags
		fmt.Printf("[LOG] Добавлено тегов: %d\n", added)
		for i := s.prevTags; i < currentTags; i++ {
			fmt.Printf("  Tag: ID=%v, Name=%s\n", tags[i].ID(), tags[i].(*model.Tag).Name())
		}
	}

	// Обновление предыдущих размеров для следующего сравнения
	s.prevNotes = currentNotes
	s.prevUsers = currentUsers
	s.prevSessions = currentSessions
	s.prevTags = currentTags
}
