package handlers

import (
	"MainProject/internal/model"
	"MainProject/internal/service"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "MainProject/cmd/NotesApp/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// LoginRequest структура для запроса авторизации
// @Description Структура для передачи учетных данных
type LoginRequest struct {
	Login    string `json:"login" example:"admin" validate:"required,min=3,max=50"`
	Password string `json:"password" example:"secret" validate:"required,min=6"`
}

// SuccessResponse структура для успешного ответа с токеном
type SuccessResponse struct {
	Token string `json:"token" example:"eyJh..."`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// HandleLogin обрабатывает POST /login, проверяет credentials и выдаёт JWT
// @Summary Авторизация
// @Description Получает JWT-токен по логину и паролю из переменных окружения LOGIN и PASSWORD
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Учетные данные"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /login [post]
func HandleLogin(c *gin.Context) {
	var creds LoginRequest
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Проверка логина из переменных окружения
	if creds.Login != os.Getenv("LOGIN") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login"})
		return
	}

	// Проверка пароля из переменных окружения
	// В продакшене следует использовать хеширование (bcrypt.CompareHashAndPassword)
	if creds.Password != os.Getenv("PASSWORD") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Создание JWT токена
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": creds.Login,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // Срок действия 24 часа
		"iat":   time.Now().Unix(),                     // Время выдачи
		"iss":   "notes-app",                           // Издатель
	})

	// Подпись токена с использованием секретного ключа из переменных окружения
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// HandlePostItem создаёт новую заметку
// @Summary Создать заметку
// @Description Создаёт новую заметку с указанными полями. Требуется авторизация.
// @Tags notes
// @Accept json
// @Produce json
// @Param note body model.Note true "Данные заметки"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security bearerAuth
// @Router /api/item [post]
func HandlePostItem(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var note model.Note
		if err := c.ShouldBindJSON(&note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный JSON"})
			return
		}

		created, err := noteService.CreateNote(
			c.Request.Context(),
			service.CreateNoteInput{
				Title:    note.Title,
				Content:  note.Content,
				Tags:     note.Tags,
				IsPublic: note.IsPublic,
			},
		)
		if err != nil {
			switch err {
			case service.ErrInvalidNote:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Поля 'title' и 'content' обязательны"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании заметки"})
			}
			return
		}

		c.JSON(http.StatusCreated, created)
	}
}

// HandleGetItem получает заметку по ID
// @Summary Получить заметку по ID
// @Description Возвращает заметку с указанным ID. Доступно без авторизации.
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "ID заметки" example("1")
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/item/{id} [get]
func HandleGetItem(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		note, err := noteService.GetNote(c.Request.Context(), id)
		if err != nil {
			if err == service.ErrNoteNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заметки"})
			return
		}

		c.JSON(http.StatusOK, note)
	}
}

// HandleGetItems получает список всех заметок
// @Summary Список заметок
// @Description Возвращает все существующие заметки. Доступно без авторизации.
// @Tags notes
// @Produce json
// @Success 200 {array} model.Note "Список заметок"
// @Router /api/items [get]
func HandleGetItems(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		notes, err := noteService.ListNotes(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списка заметок"})
			return
		}

		c.JSON(http.StatusOK, notes)
	}
}

// HandlePutItem обновляет заметку по ID
// @Summary Обновить заметку
// @Description Обновляет заметку с указанным ID. Требуется авторизация.
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "ID заметки" example("1")
// @Param note body model.Note true "Обновлённые данные заметки"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security bearerAuth
// @Router /api/item/{id} [put]
func HandlePutItem(noteService service.NoteService) gin.HandlerFunc {
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

		note, err := noteService.UpdateNote(
			c.Request.Context(),
			id,
			service.UpdateNoteInput{
				Title:    updated.Title,
				Content:  updated.Content,
				Tags:     updated.Tags,
				IsPublic: updated.IsPublic,
			},
		)
		if err != nil {
			switch err {
			case service.ErrInvalidNote:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Поля 'title' и 'content' обязательны"})
			case service.ErrNoteNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении заметки"})
			}
			return
		}

		c.JSON(http.StatusOK, note)
	}
}

// HandleDeleteItem удаляет заметку по ID
// @Summary Удалить заметку
// @Description Удаляет заметку с указанным ID. Требуется авторизация.
// @Tags notes
// @Param id path string true "ID заметки" example("1")
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security bearerAuth
// @Router /api/item/{id} [delete]
func HandleDeleteItem(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		if err := noteService.DeleteNote(c.Request.Context(), id); err != nil {
			switch err {
			case service.ErrNoteNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "Заметка не найдена"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заметки"})
			}
			return
		}

		c.Status(http.StatusNoContent) // 204 No Content — успешно удалено
	}
}
