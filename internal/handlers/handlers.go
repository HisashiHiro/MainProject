package handlers

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "MainProject/cmd/NotesApp/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// HandleLogin обрабатывает POST /login, проверяет credentials и выдаёт JWT
// @Summary Авторизация
// @Description Получает JWT-токен по логину и паролю
// @Tags auth
// @Accept  json
// @Produce  json
// @Param request body struct{Login string `json:"login"` Password string `json:"password"`} true "Credentials"
// @Success 200 {object} map[string]string "JWT token"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /login [post]
func HandleLogin(c *gin.Context) {
	var creds struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Проверка логина и пароля (в проде — из БД)
	if creds.Login != os.Getenv("LOGIN") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login"})
		return
	}

	// В проде: сравнение хеша пароля из БД с bcrypt.CompareHashAndPassword
	// Здесь упрощённо: проверяем строку
	if creds.Password != os.Getenv("PASSWORD") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Создание JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": creds.Login,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // 24 часа
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// HandlePostItem создаёт новую заметку
// @Summary Создать заметку
// @Description Создаёт новую заметку с указанными полями
// @Tags notes
// @Accept  json
// @Produce  json
// @Param note body model.Note true "Данные заметки"
// @Success 201 {object} model.Note "Созданная заметка"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security bearerAuth
// @Router /api/item [post]
func HandlePostItem(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var note model.Note
		if err := c.ShouldBindJSON(&note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный JSON"})
			return
		}

		if note.Title == "" || note.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Поля 'title' и 'content' обязательны"})
			return
		}

		note.CreatedAt = time.Now()
		note.UpdatedAt = time.Now()
		note.IsGenerated = false

		id := repo.AddNote(&note)
		note.SetID(id)

		if err := repo.SaveNoteToCSV(&note); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении в CSV"})
			return
		}

		c.JSON(http.StatusCreated, note)
	}
}

// HandleGetItem получает заметку по ID
// @Summary Получить заметку по ID
// @Description Возвращает заметку с указанным ID
// @ID get-item-by-id
// @Accept  json
// @Produce  json
// @Param id path string true "ID заметки"
// @Success 200 {object} model.Note "Заметка"
// @Failure 404 {object} map[string]string "Not Found"
// @Router /api/item/{id} [get]
func HandleGetItem(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		note, found := repo.FindNoteById(id)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
			return
		}

		c.JSON(http.StatusOK, note)
	}
}

// HandleGetItems получает список всех заметок
// @Summary Список заметок
// @Description Возвращает все существующие заметки
// @Tags notes
// @Produce json
// @Success 200 {array} model.Note "Список заметок"
// @Router /api/items [get]
func HandleGetItems(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		notes := repo.GetNotes()
		result := make([]model.Note, len(notes))
		for i, entity := range notes {
			result[i] = *entity.(*model.Note)
		}

		c.JSON(http.StatusOK, result)
	}
}

// HandlePutItem обновляет заметку по ID
// @Summary Обновить заметку
// @Description Обновляет заметку с указанным ID
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "ID заметки"
// @Param note body model.Note true "Обновлённые данные заметки"
// @Success 200 {object} model.Note "Обновлённая заметка"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 404 {object} map[string]string "Not Found"
// @Security bearerAuth
// @Router /api/item/{id} [put]
func HandlePutItem(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		var updated model.Note
		if err := c.ShouldBindJSON(&updated); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный JSON"})
			return
		}

		if updated.Title == "" || updated.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Поля 'title' и 'content' обязательны"})
			return
		}

		updated.SetID(id) // Используем метод
		updated.UpdatedAt = time.Now()

		if !repo.UpdateNote(id, &updated) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
			return
		}

		c.JSON(http.StatusOK, updated)
	}
}

// HandleDeleteItem удаляет заметку по ID
// @Summary Удалить заметку
// @Description Удаляет заметку с указанным ID
// @Tags notes
// @Param id path string true "ID заметки"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 404 {object} map[string]string "Not Found"
// @Security bearerAuth
// @Router /api/item/{id} [delete]
func HandleDeleteItem(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		if !repo.DeleteNote(id) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
			return
		}

		c.Status(http.StatusNoContent) // 204 No Content — успешно удалено
	}
}
