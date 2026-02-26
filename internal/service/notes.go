package service

import (
	"MainProject/internal/model"
	"context"
	"errors"
	"time"
)

// NoteRepository описывает только ту часть репозитория,
// которая нужна бизнес-логике заметок
type NoteRepository interface {
	AddNote(note *model.Note) int64
	FindNoteById(id int64) (*model.Note, bool)
	UpdateNote(id int64, updated *model.Note) bool
	DeleteNote(id int64) bool
	GetNotes() []model.Entity
	SaveNoteToCSV(note *model.Note) error
	DeleteNoteFromCSV(id int64) error
}

// Ошибки доменного уровня для заметок
var (
	ErrNoteNotFound = errors.New("note not found")
	ErrInvalidNote  = errors.New("invalid note")
)

// NoteService описывает бизнес‑операции над заметками
// независимо от транспортного слоя (HTTP/gRPC)
type NoteService interface {
	CreateNote(ctx context.Context, input CreateNoteInput) (*model.Note, error)
	GetNote(ctx context.Context, id int64) (*model.Note, error)
	ListNotes(ctx context.Context) ([]*model.Note, error)
	UpdateNote(ctx context.Context, id int64, input UpdateNoteInput) (*model.Note, error)
	DeleteNote(ctx context.Context, id int64) error
}

// CreateNoteInput — входные данные для создания заметки
type CreateNoteInput struct {
	Title    string
	Content  string
	Tags     []string
	IsPublic bool
}

// UpdateNoteInput — входные данные для обновления заметки
type UpdateNoteInput struct {
	Title    string
	Content  string
	Tags     []string
	IsPublic bool
}

type noteServiceImpl struct {
	repo NoteRepository
}

// NewNoteService создаёт реализацию сервиса заметок,
// использующую переданный репозиторий
func NewNoteService(repo NoteRepository) NoteService {
	return &noteServiceImpl{repo: repo}
}

// validateNoteFields проверяет обязательные поля заметки
func validateNoteFields(title, content string) error {
	if title == "" || content == "" {
		return ErrInvalidNote
	}
	return nil
}

// CreateNote создаёт новую заметку, устанавливает временные метки,
// генерирует ID через репозиторий и сохраняет в CSV
func (s *noteServiceImpl) CreateNote(ctx context.Context, input CreateNoteInput) (*model.Note, error) {
	if err := validateNoteFields(input.Title, input.Content); err != nil {
		return nil, err
	}

	note := model.NewNote(input.Title, input.Content, input.Tags, input.IsPublic)
	// Заметка создана пользователем, а не планировщиком
	note.IsGenerated = false

	// Уточняем временные метки, чтобы они точно были установлены сейчас
	now := time.Now()
	note.CreatedAt = now
	note.UpdatedAt = now

	id := s.repo.AddNote(note)
	note.SetID(id)

	if err := s.repo.SaveNoteToCSV(note); err != nil {
		return nil, err
	}

	return note, nil
}

// GetNote возвращает заметку по ID
func (s *noteServiceImpl) GetNote(ctx context.Context, id int64) (*model.Note, error) {
	note, found := s.repo.FindNoteById(id)
	if !found {
		return nil, ErrNoteNotFound
	}
	return note, nil
}

// ListNotes возвращает все заметки в виде слайса доменных моделей
func (s *noteServiceImpl) ListNotes(ctx context.Context) ([]*model.Note, error) {
	entities := s.repo.GetNotes()
	result := make([]*model.Note, 0, len(entities))
	for _, e := range entities {
		if note, ok := e.(*model.Note); ok {
			result = append(result, note)
		}
	}
	return result, nil
}

// UpdateNote обновляет заметку по ID
func (s *noteServiceImpl) UpdateNote(ctx context.Context, id int64, input UpdateNoteInput) (*model.Note, error) {
	if err := validateNoteFields(input.Title, input.Content); err != nil {
		return nil, err
	}

	// Проверим, что заметка существует
	existing, found := s.repo.FindNoteById(id)
	if !found {
		return nil, ErrNoteNotFound
	}

	// Обновляем поля
	existing.SetTitle(input.Title)
	existing.SetContent(input.Content)
	existing.Tags = input.Tags
	existing.SetPublic(input.IsPublic)
	existing.UpdatedAt = time.Now()

	if !s.repo.UpdateNote(id, existing) {
		return nil, ErrNoteNotFound
	}

	return existing, nil
}

// DeleteNote удаляет заметку и связанную с ней запись в CSV
func (s *noteServiceImpl) DeleteNote(ctx context.Context, id int64) error {
	if !s.repo.DeleteNote(id) {
		return ErrNoteNotFound
	}

	if err := s.repo.DeleteNoteFromCSV(id); err != nil {
		return err
	}

	return nil
}
