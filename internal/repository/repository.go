package repository

import (
	"MainProject/internal/model"
)

// Repository — репозиторий для хранения сущностей разных типов.
type Repository struct {
	notes    []model.Entity
	users    []model.Entity
	sessions []model.Entity
	tags     []model.Entity
}

// NewRepository создаёт новый репозиторий.
func NewRepository() *Repository {
	return &Repository{
		notes:    make([]model.Entity, 0),
		users:    make([]model.Entity, 0),
		sessions: make([]model.Entity, 0),
		tags:     make([]model.Entity, 0),
	}
}

// SaveEntities принимает срез сущностей и распределяет их по соответствующим слайсам.
func (r *Repository) SaveEntities(entities []model.Entity) {
	for _, entity := range entities {
		switch v := entity.(type) {
		case *model.Note:
			r.notes = append(r.notes, v)
		case *model.User:
			r.users = append(r.users, v)
		case *model.Session:
			r.sessions = append(r.sessions, v)
		case *model.Tag:
			r.tags = append(r.tags, v)
		default:
			// Обработать неизвестный тип, логировать
		}
	}
}

// Методы для получения данных (для примера)
func (r *Repository) GetNotes() []model.Entity {
	return r.notes
}

func (r *Repository) GetUsers() []model.Entity {
	return r.users
}

func (r *Repository) GetSession() []model.Entity {
	return r.sessions
}

func (r *Repository) GetTags() []model.Entity {
	return r.tags
}
