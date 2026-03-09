package service

import (
	"MainProject/internal/model"
	"context"
	"errors"
	"testing"
	"time"
)

// fakeNoteRepository — простая in-memory реализация NoteRepository
// для unit‑тестов сервиса заметок.
// Позволяет тестировать бизнес‑логику без реального хранилища
type fakeNoteRepository struct {
	notes  map[int64]*model.Note
	tags   map[int64][]string
	nextID int64
}

// newFakeNoteRepository создаёт пустой фейковый репозиторий заметок
func newFakeNoteRepository() *fakeNoteRepository {
	return &fakeNoteRepository{
		notes:  make(map[int64]*model.Note),
		tags:   make(map[int64][]string),
		nextID: 1,
	}
}

// CreateNote эмулирует сохранение заметки с авто‑инкрементным ID
func (r *fakeNoteRepository) CreateNote(ctx context.Context, note *model.Note) (int64, error) {
	id := r.nextID
	r.nextID++
	note.ID = id
	// Сохраняем копию, чтобы избежать внешних изменений
	noteCopy := *note
	r.notes[id] = &noteCopy
	// Если есть теги, сохраняем их отдельно (имитируем связи)
	if len(note.Tags) > 0 {
		tagsCopy := make([]string, len(note.Tags))
		copy(tagsCopy, note.Tags)
		r.tags[id] = tagsCopy
	}
	return id, nil
}

func (r *fakeNoteRepository) CreateNoteWithTags(ctx context.Context, note *model.Note, tags []string) (int64, error) {
	id := r.nextID
	r.nextID++
	note.ID = id
	// Сохраняем заметку
	noteCopy := *note
	noteCopy.Tags = tags // устанавливаем теги в копии
	r.notes[id] = &noteCopy
	// Сохраняем теги отдельно
	if len(tags) > 0 {
		tagsCopy := make([]string, len(tags))
		copy(tagsCopy, tags)
		r.tags[id] = tagsCopy
	}
	return id, nil
}

// FindNoteByID возвращает заметку по ID, если она существует
func (r *fakeNoteRepository) FindNoteByID(ctx context.Context, id int64) (*model.Note, bool, error) {
	n, ok := r.notes[id]
	if !ok {
		return nil, false, nil
	}
	// Создаём копию, чтобы не изменять оригинал в хранилище
	noteCopy := *n
	noteCopy.Tags = r.tags[id]
	return &noteCopy, true, nil
}

// UpdateNote заменяет сохранённую заметку новой версией
func (r *fakeNoteRepository) UpdateNote(ctx context.Context, note *model.Note) (bool, error) {
	id := note.ID
	if _, ok := r.notes[id]; !ok {
		return false, nil
	}
	// Обновляем заметку
	noteCopy := *note
	r.notes[id] = &noteCopy
	// Обновляем теги
	if len(note.Tags) > 0 {
		tagsCopy := make([]string, len(note.Tags))
		copy(tagsCopy, note.Tags)
		r.tags[id] = tagsCopy
	} else {
		delete(r.tags, id)
	}
	return true, nil
}

// DeleteNote удаляет заметку из in-memory хранилища
func (r *fakeNoteRepository) DeleteNote(ctx context.Context, id int64) (bool, error) {
	if _, ok := r.notes[id]; !ok {
		return false, nil
	}
	delete(r.notes, id)
	delete(r.tags, id)
	return true, nil
}

// ListNotes возвращает все заметки
func (r *fakeNoteRepository) ListNotes(ctx context.Context) ([]*model.Note, error) {
	res := make([]*model.Note, 0, len(r.notes))
	for id, n := range r.notes {
		noteCopy := *n
		noteCopy.Tags = r.tags[id]
		res = append(res, &noteCopy)
	}
	return res, nil
}

func TestNoteService_CreateNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		input   CreateNoteInput
		wantErr error
	}{
		{
			name: "ok - valid note without tags",
			input: CreateNoteInput{
				Title:    "Title",
				Content:  "Content",
				IsPublic: true,
			},
		},
		{
			name: "ok - valid note with tags",
			input: CreateNoteInput{
				Title:    "Title",
				Content:  "Content",
				Tags:     []string{"tag1", "tag2"},
				IsPublic: true,
			},
		},
		{
			name: "error - empty title",
			input: CreateNoteInput{
				Title:    "",
				Content:  "Content",
				IsPublic: true,
			},
			wantErr: ErrInvalidNote,
		},
		{
			name: "error - empty content",
			input: CreateNoteInput{
				Title:    "Title",
				Content:  "",
				IsPublic: true,
			},
			wantErr: ErrInvalidNote,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeNoteRepository()
			svc := NewNoteService(repo)

			note, err := svc.CreateNote(ctx, tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if note == nil {
				t.Fatalf("expected note, got nil")
			}
			if note.Title != tt.input.Title || note.Content != tt.input.Content {
				t.Errorf("note fields mismatch")
			}
			if note.ID == 0 {
				t.Errorf("expected ID to be set")
			}
			if note.CreatedAt.IsZero() || note.UpdatedAt.IsZero() {
				t.Errorf("expected timestamps to be set")
			}
			// Проверяем теги
			if len(note.Tags) != len(tt.input.Tags) {
				t.Errorf("tags count mismatch: expected %d, got %d", len(tt.input.Tags), len(note.Tags))
			}
			for i, tag := range tt.input.Tags {
				if note.Tags[i] != tag {
					t.Errorf("tag mismatch at %d: expected %s, got %s", i, tag, note.Tags[i])
				}
			}
		})
	}
}
func TestNoteService_GetNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	// Подготовка данных
	created, err := svc.CreateNote(ctx, CreateNoteInput{
		Title:   "Existing",
		Content: "Content",
		Tags:    []string{"tagA", "tagB"},
	})
	if err != nil {
		t.Fatalf("setup CreateNote error: %v", err)
	}

	tests := []struct {
		name    string
		id      int64
		wantErr error
	}{
		{
			name: "ok - found",
			id:   created.ID,
		},
		{
			name:    "error - not found",
			id:      999,
			wantErr: ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			note, err := svc.GetNote(ctx, tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if note != nil {
					t.Fatalf("expected nil note, got %+v", note)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if note == nil || note.ID != created.ID {
				t.Fatalf("unexpected note returned")
			}
			// Проверяем теги
			if len(note.Tags) != 2 || note.Tags[0] != "tagA" || note.Tags[1] != "tagB" {
				t.Errorf("tags not restored correctly: %v", note.Tags)
			}
		})
	}
}

func TestNoteService_ListNotes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	// Нет заметок
	notes, err := svc.ListNotes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(notes))
	}

	// Добавим две заметки с тегами
	_, _ = svc.CreateNote(ctx, CreateNoteInput{Title: "n1", Content: "c1", Tags: []string{"work"}})
	_, _ = svc.CreateNote(ctx, CreateNoteInput{Title: "n2", Content: "c2", Tags: []string{"personal", "urgent"}})

	notes, err = svc.ListNotes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	// Проверим, что теги загружены
	for _, n := range notes {
		if n.Title == "n1" && (len(n.Tags) != 1 || n.Tags[0] != "work") {
			t.Errorf("note n1 tags incorrect: %v", n.Tags)
		}
		if n.Title == "n2" && (len(n.Tags) != 2 || n.Tags[0] != "personal" || n.Tags[1] != "urgent") {
			t.Errorf("note n2 tags incorrect: %v", n.Tags)
		}
	}
}

func TestNoteService_UpdateNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(*fakeNoteRepository) int64
		idOffset int64
		input    UpdateNoteInput
		wantErr  error
	}{
		{
			name: "ok - update existing without tags",
			setup: func(r *fakeNoteRepository) int64 {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, CreateNoteInput{
					Title:   "old",
					Content: "old content",
				})
				return n.ID
			},
			input: UpdateNoteInput{
				Title:   "new",
				Content: "new content",
			},
		},
		{
			name: "ok - update existing with tags",
			setup: func(r *fakeNoteRepository) int64 {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, CreateNoteInput{
					Title:   "old",
					Content: "old content",
					Tags:    []string{"oldtag"},
				})
				return n.ID
			},
			input: UpdateNoteInput{
				Title:   "new",
				Content: "new content",
				Tags:    []string{"newtag1", "newtag2"},
			},
		},
		{
			name: "error - invalid fields",
			setup: func(r *fakeNoteRepository) int64 {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, CreateNoteInput{
					Title:   "old",
					Content: "old content",
				})
				return n.ID
			},
			input: UpdateNoteInput{
				Title:   "",
				Content: "content",
			},
			wantErr: ErrInvalidNote,
		},
		{
			name: "error - not found",
			setup: func(r *fakeNoteRepository) int64 {
				return 123
			},
			idOffset: 0,
			input: UpdateNoteInput{
				Title:   "new",
				Content: "new content",
			},
			wantErr: ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeNoteRepository()
			baseID := tt.setup(repo)
			svc := NewNoteService(repo)

			id := baseID + tt.idOffset

			note, err := svc.UpdateNote(ctx, id, tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if note != nil {
					t.Fatalf("expected nil note, got %+v", note)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if note == nil {
				t.Fatalf("expected note, got nil")
			}
			if note.Title != tt.input.Title || note.Content != tt.input.Content {
				t.Errorf("note fields were not updated")
			}
			// Проверяем теги
			if len(note.Tags) != len(tt.input.Tags) {
				t.Errorf("tags count mismatch: expected %d, got %d", len(tt.input.Tags), len(note.Tags))
			}
			for i, tag := range tt.input.Tags {
				if note.Tags[i] != tag {
					t.Errorf("tag mismatch at %d: expected %s, got %s", i, tag, note.Tags[i])
				}
			}
		})
	}
}

func TestNoteService_DeleteNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(*fakeNoteRepository) int64
		idOffset int64
		wantErr  error
	}{
		{
			name: "ok - delete existing",
			setup: func(r *fakeNoteRepository) int64 {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, CreateNoteInput{
					Title:   "t",
					Content: "c",
				})
				return n.ID
			},
		},
		{
			name: "error - not found",
			setup: func(r *fakeNoteRepository) int64 {
				return 999
			},
			wantErr: ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeNoteRepository()
			id := tt.setup(repo)
			svc := NewNoteService(repo)

			err := svc.DeleteNote(ctx, id+tt.idOffset)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Проверяем, что заметка действительно удалена
			_, found, _ := repo.FindNoteByID(ctx, id)
			if found {
				t.Errorf("note still exists after delete")
			}
		})
	}
}

// Дополнительный тест, чтобы убедиться, что timestamps действительно обновляются при изменении
func TestNoteService_UpdateNote_UpdatesTimestamp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	n, err := svc.CreateNote(ctx, CreateNoteInput{
		Title:   "t",
		Content: "c",
	})
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}

	oldUpdatedAt := n.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	updated, err := svc.UpdateNote(ctx, n.ID, UpdateNoteInput{
		Title:   "t2",
		Content: "c2",
	})
	if err != nil {
		t.Fatalf("UpdateNote error: %v", err)
	}

	if !updated.UpdatedAt.After(oldUpdatedAt) {
		t.Fatalf("expected UpdatedAt to be changed")
	}
}
