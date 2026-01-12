package repository

import (
	"MainProject/internal/model"
	"context"
	"sync"
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

	// Сохранение контекста для отслеживания завершения
	ctx context.Context
}

// NewRepository создаёт новый репозиторий.
func NewRepository(ctx context.Context) *Repository {
	repo := &Repository{
		notes:     make([]model.Entity, 0),
		users:     make([]model.Entity, 0),
		sessions:  make([]model.Entity, 0),
		tags:      make([]model.Entity, 0),
		inputChan: make(chan []model.Entity, 100), // Буферизованный канал

		// Сохранение переданного контекста
		ctx: ctx,
	}

	// Запуск горутины для обработки входящих сущностей
	go repo.processEntities()

	return repo
}

// Метод для получения канала для возможности сервисом отправлять данные
func (r *Repository) InputChannel() chan<- []model.Entity {
	return r.inputChan
}

// processEntities — горутина, непрерывно обрабатывающая входящие сущности из канала inputChan
func (r *Repository) processEntities() {
	for {
		// Одновременное ожидание:
		// 1. Данных из канала inputChan
		// 2. Сигнала отмены из контекста (ctx.Done())
		select {
		// Бесконечный цикл чтения из канала inputChan
		case entities, ok := <-r.inputChan:
			// Если канал закрыт (ok == false), завершаем работу
			if !ok {
				return
			}
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
		case <-r.ctx.Done():
			// Получен сигнал отмены (вызван cancel())
			// Закрытие канала inputChan
			close(r.inputChan)
			return // Завершение горутины
		}
	}
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
