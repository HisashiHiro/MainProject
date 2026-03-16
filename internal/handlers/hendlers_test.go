package handlers_test

import (
	"MainProject/internal/handlers"
	"MainProject/internal/model"
	"MainProject/internal/service"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNoteService — ручная реализация интерфейса service.NoteService для тестов
type mockNoteService struct {
	createNoteFunc func(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error)
	getNoteFunc    func(ctx context.Context, userID int64, id int64) (*model.Note, error)
	listNotesFunc  func(ctx context.Context, userID int64) ([]*model.Note, error)
	updateNoteFunc func(ctx context.Context, userID int64, id int64, input service.UpdateNoteInput) (*model.Note, error)
	deleteNoteFunc func(ctx context.Context, userID int64, id int64) error
}

func (m *mockNoteService) CreateNote(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error) {
	return m.createNoteFunc(ctx, userID, input)
}
func (m *mockNoteService) GetNote(ctx context.Context, userID int64, id int64) (*model.Note, error) {
	return m.getNoteFunc(ctx, userID, id)
}
func (m *mockNoteService) ListNotes(ctx context.Context, userID int64) ([]*model.Note, error) {
	return m.listNotesFunc(ctx, userID)
}
func (m *mockNoteService) UpdateNote(ctx context.Context, userID int64, id int64, input service.UpdateNoteInput) (*model.Note, error) {
	return m.updateNoteFunc(ctx, userID, id, input)
}
func (m *mockNoteService) DeleteNote(ctx context.Context, userID int64, id int64) error {
	return m.deleteNoteFunc(ctx, userID, id)
}

// setupTestRouter создаёт тестовый роутер без middleware аутентификации,
// но с возможностью установки userID в контекст через параметр.
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHandlePostItem(t *testing.T) {
	// Создаём мок сервиса
	mockSvc := &mockNoteService{
		createNoteFunc: func(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error) {
			assert.Equal(t, int64(123), userID)
			assert.Equal(t, "Test Title", input.Title)
			assert.Equal(t, "Test Content", input.Content)
			return &model.Note{
				ID:        1,
				UserID:    userID,
				Title:     input.Title,
				Content:   input.Content,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	// Регистрируем хендлер
	handler := handlers.HandlePostItem(mockSvc)

	r := setupTestRouter()
	r.POST("/api/item", func(c *gin.Context) {
		// Устанавливаем userID в контекст (как это делает JWTMiddleware)
		c.Set("userID", int64(123))
		handler(c)
	})

	// Формируем запрос
	reqBody := map[string]interface{}{
		"title":   "Test Title",
		"content": "Test Content",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/item", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp model.Note
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "Test Title", resp.Title)
}

func TestHandleGetItem(t *testing.T) {
	mockSvc := &mockNoteService{
		getNoteFunc: func(ctx context.Context, userID int64, id int64) (*model.Note, error) {
			assert.Equal(t, int64(123), userID)
			assert.Equal(t, int64(42), id)
			return &model.Note{
				ID:        42,
				UserID:    userID,
				Title:     "Found",
				Content:   "Content",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := handlers.HandleGetItem(mockSvc)

	r := setupTestRouter()
	r.GET("/api/item/:id", func(c *gin.Context) {
		c.Set("userID", int64(123))
		handler(c)
	})

	req, _ := http.NewRequest("GET", "/api/item/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.Note
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Found", resp.Title)
}

// Аналогичные тесты для Put, Delete, List...
