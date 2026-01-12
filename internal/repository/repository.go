package repository

import (
	"MainProject/internal/model"
	"fmt"
	"sync"
	"time"
)

// Repository — репозиторий для хранения сущностей разных типов
type Repository struct {
	notes    []model.Entity
	users    []model.Entity
	sessions []model.Entity
	tags     []model.Entity

	// Мьютексы для конкурентного доступа
	muNotes    sync.Mutex
	muUsers    sync.Mutex
	muSessions sync.Mutex
	muTags     sync.Mutex

	// Канал для приёма сущностей
	inputChan chan []model.Entity

	// Поля для логирования изменений
	// Предыдущие размеры слайсов (для отслеживания изменений)
	prevNotes    int
	prevUsers    int
	prevSessions int
	prevTags     int
}

// NewRepository создаёт новый репозиторий.
func NewRepository() *Repository {
	repo := &Repository{
		notes:     make([]model.Entity, 0),
		users:     make([]model.Entity, 0),
		sessions:  make([]model.Entity, 0),
		tags:      make([]model.Entity, 0),
		inputChan: make(chan []model.Entity, 100), // Буферизованный канал
		// Инициализация предыдущих размеров слайсов нулями (начальное состояние)
		prevNotes:    0,
		prevUsers:    0,
		prevSessions: 0,
		prevTags:     0,
	}

	// Запуск горутины для обработки входящих сущностей
	go repo.processEntities()

	// Запускаем горутину для логирования изменений (каждые 200 мс)
	go repo.runLogger(200 * time.Millisecond)

	return repo
}

// Метод для получения канала для возможности сервисом отправлять данные
func (r *Repository) InputChannel() chan<- []model.Entity {
	return r.inputChan
}

// processEntities — горутина, непрерывно обрабатывающая входящие сущности из канала inputChan
func (r *Repository) processEntities() {
	// Бесконечный цикл чтения из канала inputChan
	// Завершается автоматически при закрытии канала (остановке приложения)
	for entities := range r.inputChan {
		// Обработка каждой сущности в полученной группе
		for _, entity := range entities {
			// Определение конкретного типа сущности через type switch
			// Что позволяет корректно добавить объект в соответствующий слайс
			switch v := entity.(type) {
			case *model.Note:
				// 1. Блокировка доступа к слайсу notes на время модификации
				// 2. Добавление сущности в слайс
				// 3. Снятие блокировки
				r.muNotes.Lock()
				r.notes = append(r.notes, v)
				r.muNotes.Unlock()
			case *model.User:
				r.muUsers.Lock()
				r.users = append(r.users, v)
				r.muUsers.Unlock()
			case *model.Session:
				r.muSessions.Lock()
				r.sessions = append(r.sessions, v)
				r.muSessions.Unlock()
			case *model.Tag:
				r.muTags.Lock()
				r.tags = append(r.tags, v)
				r.muTags.Unlock()
			default:
				// Тип сущности не распознан (логирование ошибки)
			}
		}
	}
}

// runLogger — горутина, которая периодически проверяет изменения в слайсах и логирует новые элементы.
// Параметр interval определяет, как часто проверять изменения
// Использует Ticker для периодического вызова метода logChanges()
func (r *Repository) runLogger(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		r.logChanges()
	}
}

// logChanges выполняет:
// 1. Получение текущих данных из репозитория (с мьютексами)
// 2. Сравнение текущих размеров слайсов с предыдущими значениями
// 3. Логирование только новых элементов (которые появились с последнего вызова)
// 4. Обновление предыдущих размеров для следующего цикла
func (r *Repository) logChanges() {
	// Получение текущих данных из репозитория (методы уже содержат мьютексы)
	notes := r.GetNotes()
	users := r.GetUsers()
	sessions := r.GetSessions()
	tags := r.GetTags()

	// Текущие размеры слайсов
	currentNotes := len(notes)
	currentUsers := len(users)
	currentSessions := len(sessions)
	currentTags := len(tags)

	// Проверка и логирование новых заметок
	// Проверка изменения размеров слайсов с последнего раза
	if currentNotes > r.prevNotes {
		added := currentNotes - r.prevNotes
		fmt.Printf("[LOG] Добавлено заметок: %d\n", added)
		// Логирование только новых элементов (от prevNotes до currentNotes)
		for i := r.prevNotes; i < currentNotes; i++ {
			fmt.Printf("  Note: ID=%v, Title=%s\n", notes[i].ID(), notes[i].(*model.Note).Title())
		}
	}

	// Проверка и логирование новых пользователей
	if currentUsers > r.prevUsers {
		added := currentUsers - r.prevUsers
		fmt.Printf("[LOG] Добавлено пользователей: %d\n", added)
		for i := r.prevUsers; i < currentUsers; i++ {
			fmt.Printf("  User: ID=%v, Username=%s\n", users[i].ID(), users[i].(*model.User).Username())
		}
	}

	// Проверка и логирование новых сессий
	if currentSessions > r.prevSessions {
		added := currentSessions - r.prevSessions
		fmt.Printf("[LOG] Добавлено сессий: %d\n", added)
		for i := r.prevSessions; i < currentSessions; i++ {
			fmt.Printf("  Session: ID=%v, UserID=%d\n", sessions[i].ID(), sessions[i].(*model.Session).UserID())
		}
	}

	// Проверка и логирование новых тегов
	if currentTags > r.prevTags {
		added := currentTags - r.prevTags
		fmt.Printf("[LOG] Добавлено тегов: %d\n", added)
		for i := r.prevTags; i < currentTags; i++ {
			fmt.Printf("  Tag: ID=%v, Name=%s\n", tags[i].ID(), tags[i].(*model.Tag).Name())
		}
	}

	// Обновление предыдущих размеров для следующего сравнения
	r.prevNotes = currentNotes
	r.prevUsers = currentUsers
	r.prevSessions = currentSessions
	r.prevTags = currentTags
}

// Методы для безопасного чтения (с мьютексами)
func (r *Repository) GetNotes() []model.Entity {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()
	return r.notes
}

func (r *Repository) GetUsers() []model.Entity {
	r.muUsers.Lock()
	defer r.muUsers.Unlock()
	return r.users
}

func (r *Repository) GetSessions() []model.Entity {
	r.muSessions.Lock()
	defer r.muSessions.Unlock()
	return r.sessions
}

func (r *Repository) GetTags() []model.Entity {
	r.muTags.Lock()
	defer r.muTags.Unlock()
	return r.tags
}
