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
	notes  map[int64]map[int64]*model.Note
	tags   map[int64]map[int64][]string
	nextID map[int64]int64
}

// newFakeNoteRepository создаёт пустой фейковый репозиторий заметок
func newFakeNoteRepository() *fakeNoteRepository {
	return &fakeNoteRepository{
		notes:  make(map[int64]map[int64]*model.Note),
		tags:   make(map[int64]map[int64][]string),
		nextID: make(map[int64]int64),
	}
}

// getNextID возвращает следующий ID для указанного пользователя
func (r *fakeNoteRepository) getNextID(userID int64) int64 {
	r.nextID[userID]++
	return r.nextID[userID]
}

// CreateNote сохраняет заметку для указанного пользователя
func (r *fakeNoteRepository) CreateNote(ctx context.Context, note *model.Note, userID int64) (int64, error) {
	id := r.getNextID(userID)
	note.ID = id
	note.UserID = userID

	// Инициализируем мапы для пользователя, если нужно
	if _, ok := r.notes[userID]; !ok {
		r.notes[userID] = make(map[int64]*model.Note)
		r.tags[userID] = make(map[int64][]string)
	}

	// Сохраняем копию
	noteCopy := *note
	r.notes[userID][id] = &noteCopy
	if len(note.Tags) > 0 {
		tagsCopy := make([]string, len(note.Tags))
		copy(tagsCopy, note.Tags)
		r.tags[userID][id] = tagsCopy
	}
	return id, nil
}

// CreateNoteWithTags создаёт заметку с тегами для указанного пользователя
func (r *fakeNoteRepository) CreateNoteWithTags(ctx context.Context, note *model.Note, tags []string, userID int64) (int64, error) {
	note.Tags = tags
	return r.CreateNote(ctx, note, userID)
}

// FindNoteByID возвращает заметку по ID, если она существует
func (r *fakeNoteRepository) FindNoteByID(ctx context.Context, id int64, userID int64) (*model.Note, bool, error) {
	userNotes, ok := r.notes[userID]
	if !ok {
		return nil, false, nil
	}
	n, ok := userNotes[id]
	if !ok {
		return nil, false, nil
	}
	// Создаём копию, чтобы не изменять оригинал в хранилище
	noteCopy := *n
	noteCopy.Tags = r.tags[userID][id]
	return &noteCopy, true, nil
}

// UpdateNote заменяет сохранённую заметку новой версией
func (r *fakeNoteRepository) UpdateNote(ctx context.Context, note *model.Note, userID int64) (bool, error) {
	userNotes, ok := r.notes[userID]
	if !ok {
		return false, nil
	}
	if _, ok := userNotes[note.ID]; !ok {
		return false, nil
	}
	// Обновляем заметку
	noteCopy := *note
	r.notes[userID][note.ID] = &noteCopy
	// Обновляем теги
	if len(note.Tags) > 0 {
		tagsCopy := make([]string, len(note.Tags))
		copy(tagsCopy, note.Tags)
		r.tags[userID][note.ID] = tagsCopy
	} else {
		delete(r.tags[userID], note.ID)
	}
	return true, nil
}

// DeleteNote удаляет заметку из in-memory хранилища
func (r *fakeNoteRepository) DeleteNote(ctx context.Context, id int64, userID int64) (bool, error) {
	userNotes, ok := r.notes[userID]
	if !ok {
		return false, nil
	}
	if _, ok := userNotes[id]; !ok {
		return false, nil
	}
	delete(userNotes, id)
	delete(r.tags[userID], id)
	return true, nil
}

// ListNotes возвращает все заметки
func (r *fakeNoteRepository) ListNotes(ctx context.Context, userID int64) ([]*model.Note, error) {
	userNotes, ok := r.notes[userID]
	if !ok {
		return []*model.Note{}, nil
	}
	res := make([]*model.Note, 0, len(userNotes))
	for id, n := range userNotes {
		noteCopy := *n
		noteCopy.Tags = r.tags[userID][id]
		res = append(res, &noteCopy)
	}
	return res, nil
}

func TestNoteService_CreateNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(1) // фиксированный пользователь для тестов

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

		{
			name: "ok - with expires_at",
			input: CreateNoteInput{
				Title:    "Title",
				Content:  "Content",
				IsPublic: true,
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(24 * time.Hour)
					return &t
				}(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeNoteRepository()
			svc := NewNoteService(repo)

			note, err := svc.CreateNote(ctx, userID, tt.input)

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
			if note.UserID != userID {
				t.Errorf("expected UserID %d, got %d", userID, note.UserID)
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
	userID := int64(1)
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	// Подготовка данных
	created, err := svc.CreateNote(ctx, userID, CreateNoteInput{
		Title:   "Existing",
		Content: "Content",
		Tags:    []string{"tagA", "tagB"},
	})
	if err != nil {
		t.Fatalf("setup CreateNote error: %v", err)
	}

	tests := []struct {
		name    string
		userID  int64
		id      int64
		wantErr error
	}{
		{
			name:   "ok - found",
			userID: userID,
			id:     created.ID,
		},
		{
			name:    "error - not found (wrong user)",
			userID:  999,
			id:      created.ID,
			wantErr: ErrNoteNotFound,
		},
		{
			name:    "error - not found (wrong id)",
			userID:  userID,
			id:      999,
			wantErr: ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			note, err := svc.GetNote(ctx, tt.userID, tt.id)
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
			if note.UserID != tt.userID {
				t.Errorf("expected UserID %d, got %d", tt.userID, note.UserID)
			}
			if len(note.Tags) != 2 || note.Tags[0] != "tagA" || note.Tags[1] != "tagB" {
				t.Errorf("tags not restored correctly: %v", note.Tags)
			}
		})
	}
}

func TestNoteService_ListNotes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID1 := int64(1)
	userID2 := int64(2)
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	// Нет заметок у userID1
	notes, err := svc.ListNotes(ctx, userID1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes for user1, got %d", len(notes))
	}

	// Создаём заметки для двух пользователей
	_, _ = svc.CreateNote(ctx, userID1, CreateNoteInput{Title: "u1n1", Content: "c1", Tags: []string{"work"}})
	_, _ = svc.CreateNote(ctx, userID1, CreateNoteInput{Title: "u1n2", Content: "c2", Tags: []string{"personal", "urgent"}})
	_, _ = svc.CreateNote(ctx, userID2, CreateNoteInput{Title: "u2n1", Content: "c3"})

	notes1, err := svc.ListNotes(ctx, userID1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes1) != 2 {
		t.Fatalf("expected 2 notes for user1, got %d", len(notes1))
	}
	for _, n := range notes1 {
		if n.UserID != userID1 {
			t.Errorf("note belongs to wrong user: %d", n.UserID)
		}
	}

	notes2, err := svc.ListNotes(ctx, userID2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes2) != 1 {
		t.Fatalf("expected 1 note for user2, got %d", len(notes2))
	}
	if notes2[0].Title != "u2n1" {
		t.Errorf("unexpected title for user2: %s", notes2[0].Title)
	}
}

func TestNoteService_UpdateNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(1)

	tests := []struct {
		name    string
		setup   func(*fakeNoteRepository) (int64, int64) // возвращает (noteID, userID)
		input   UpdateNoteInput
		wantErr error
	}{
		{
			name: "ok - update existing",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, userID, CreateNoteInput{
					Title:   "old",
					Content: "old content",
					Tags:    []string{"oldtag"},
				})
				return n.ID, userID
			},
			input: UpdateNoteInput{
				Title:   "new",
				Content: "new content",
				Tags:    []string{"newtag1", "newtag2"},
			},
		},
		{
			name: "error - invalid fields",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, userID, CreateNoteInput{
					Title:   "old",
					Content: "old content",
				})
				return n.ID, userID
			},
			input: UpdateNoteInput{
				Title:   "",
				Content: "content",
			},
			wantErr: ErrInvalidNote,
		},
		{
			name: "error - not found (wrong user)",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, userID, CreateNoteInput{
					Title:   "old",
					Content: "old content",
				})
				return n.ID, 999
			},
			input: UpdateNoteInput{
				Title:   "new",
				Content: "new content",
			},
			wantErr: ErrNoteNotFound,
		},
		{
			name: "error - not found (wrong id)",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				return 999, userID
			},
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
			noteID, updateUserID := tt.setup(repo)
			svc := NewNoteService(repo)

			updated, err := svc.UpdateNote(ctx, updateUserID, noteID, tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if updated != nil {
					t.Fatalf("expected nil note, got %+v", updated)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated == nil {
				t.Fatalf("expected note, got nil")
			}
			if updated.Title != tt.input.Title || updated.Content != tt.input.Content {
				t.Errorf("note fields were not updated")
			}
			if updated.UserID != updateUserID {
				t.Errorf("UserID mismatch: expected %d, got %d", updateUserID, updated.UserID)
			}
			if len(updated.Tags) != len(tt.input.Tags) {
				t.Errorf("tags count mismatch: expected %d, got %d", len(tt.input.Tags), len(updated.Tags))
			}
			for i, tag := range tt.input.Tags {
				if updated.Tags[i] != tag {
					t.Errorf("tag mismatch at %d: expected %s, got %s", i, tag, updated.Tags[i])
				}
			}
		})
	}
}

func TestNoteService_DeleteNote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(1)

	tests := []struct {
		name    string
		setup   func(*fakeNoteRepository) (int64, int64)
		wantErr error
	}{
		{
			name: "ok - delete existing",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, userID, CreateNoteInput{
					Title:   "t",
					Content: "c",
				})
				return n.ID, userID
			},
		},
		{
			name: "error - not found (wrong user)",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				svc := NewNoteService(r)
				n, _ := svc.CreateNote(ctx, userID, CreateNoteInput{
					Title:   "t",
					Content: "c",
				})
				return n.ID, 999
			},
			wantErr: ErrNoteNotFound,
		},
		{
			name: "error - not found (wrong id)",
			setup: func(r *fakeNoteRepository) (int64, int64) {
				return 999, userID
			},
			wantErr: ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeNoteRepository()
			noteID, deleteUserID := tt.setup(repo)
			svc := NewNoteService(repo)

			err := svc.DeleteNote(ctx, deleteUserID, noteID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Проверяем, что заметка действительно удалена для правильного пользователя
			_, found, _ := repo.FindNoteByID(ctx, noteID, userID)
			if found {
				t.Errorf("note still exists for original user after delete")
			}
		})
	}
}

// Дополнительный тест, чтобы убедиться, что timestamps действительно обновляются при изменении
func TestNoteService_UpdateNote_UpdatesTimestamp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(1)
	repo := newFakeNoteRepository()
	svc := NewNoteService(repo)

	n, err := svc.CreateNote(ctx, userID, CreateNoteInput{
		Title:   "t",
		Content: "c",
	})
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}

	oldUpdatedAt := n.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	updated, err := svc.UpdateNote(ctx, userID, n.ID, UpdateNoteInput{
		Title:   "t2",
		Content: "c2",
	})
	if err != nil {
		t.Fatalf("UpdateNote error: %v", err)
	}

	if !updated.UpdatedAt.After(oldUpdatedAt) {
		t.Fatalf("expected UpdatedAt to be changed")
	}
	if updated.UserID != userID {
		t.Errorf("UserID changed: expected %d, got %d", userID, updated.UserID)
	}
}
