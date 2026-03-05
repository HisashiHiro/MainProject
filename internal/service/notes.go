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
	// CreateNote сохраняет заметку и возвращает сгенерированный ID
	// Реализация может использовать авто-инкремент (счётчики) или другой механизм
	CreateNote(ctx context.Context, note *model.Note) (int64, error)
	// FindNoteByID возвращает заметку и флаг found.
	FindNoteByID(ctx context.Context, id int64) (*model.Note, bool, error)
	// UpdateNote заменяет заметку целиком (по note.ID())
	UpdateNote(ctx context.Context, note *model.Note) (bool, error)
	// DeleteNote удаляет заметку по ID
	DeleteNote(ctx context.Context, id int64) (bool, error)
	// ListNotes возвращает все заметки
	ListNotes(ctx context.Context) ([]*model.Note, error)
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
// генерирует ID через репозиторий и сохраняет в хранилище (MongoDB и т.п.)
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

	id, err := s.repo.CreateNote(ctx, note)
	if err != nil {
		return nil, err
	}
	note.SetID(id)

	return note, nil
}

// GetNote возвращает заметку по ID
func (s *noteServiceImpl) GetNote(ctx context.Context, id int64) (*model.Note, error) {
	note, found, err := s.repo.FindNoteByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNoteNotFound
	}
	return note, nil
}

// ListNotes возвращает все заметки в виде слайса доменных моделей
func (s *noteServiceImpl) ListNotes(ctx context.Context) ([]*model.Note, error) {
	return s.repo.ListNotes(ctx)
}

// UpdateNote обновляет заметку по ID
func (s *noteServiceImpl) UpdateNote(ctx context.Context, id int64, input UpdateNoteInput) (*model.Note, error) {
	if err := validateNoteFields(input.Title, input.Content); err != nil {
		return nil, err
	}

	// Проверим, что заметка существует
	existing, found, err := s.repo.FindNoteByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNoteNotFound
	}

	// Обновляем поля
	existing.SetTitle(input.Title)
	existing.SetContent(input.Content)
	existing.Tags = input.Tags
	existing.SetPublic(input.IsPublic)
	existing.UpdatedAt = time.Now()

	ok, err := s.repo.UpdateNote(ctx, existing)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNoteNotFound
	}

	return existing, nil
}

// DeleteNote удаляет заметку.
func (s *noteServiceImpl) DeleteNote(ctx context.Context, id int64) error {
	ok, err := s.repo.DeleteNote(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoteNotFound
	}

	return nil
}
