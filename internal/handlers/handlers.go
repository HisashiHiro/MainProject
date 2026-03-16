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

// RegisterRequest структура для запроса регистрации
type RegisterRequest struct {
	Username string `json:"username" example:"john" validate:"required,min=3,max=50"`
	Email    string `json:"email" example:"john@example.com" validate:"required,email"`
	Password string `json:"password" example:"secret" validate:"required,min=6"`
}

// LoginRequest структура для запроса авторизации
// @Description Структура для передачи учетных данных
type LoginRequest struct {
	Login    string `json:"login" example:"admin" validate:"required"`
	Password string `json:"password" example:"secret" validate:"required"`
}

// SuccessResponse структура для успешного ответа с токеном
type SuccessResponse struct {
	Token string `json:"token" example:"eyJh..."`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// HandleRegister обрабатывает POST /api/register
// @Summary Регистрация
// @Description Создаёт нового пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные регистрации"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/register [post]
func HandleRegister(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		user, err := userService.Register(c.Request.Context(), req.Username, req.Email, req.Password)
		if err != nil {
			if err == service.ErrUserAlreadyExists {
				c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
			return
		}

		// Генерируем JWT токен для нового пользователя (опционально)
		token, err := generateJWT(user.ID, user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"token": token})
	}
}

// HandleLogin обрабатывает POST /api/login
// @Summary Авторизация
// @Description Возвращает JWT-токен по логину/email и паролю
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Учетные данные"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/login [post]
func HandleLogin(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		user, err := userService.Login(c.Request.Context(), req.Login, req.Password)
		if err != nil {
			if err == service.ErrInvalidCredentials {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
			return
		}

		token, err := generateJWT(user.ID, user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

// generateJWT создаёт JWT токен для пользователя
func generateJWT(userID int64, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"login":   username,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
		"iss":     "notes-app",
	})
	return token.SignedString([]byte(GetJWTSecret()))
}

// GetJWTSecret возвращает секрет из переменной окружения
func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

// JWTMiddleware — middleware для проверки JWT
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(GetJWTSecret()), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, ok := claims["user_id"].(float64); ok {
				c.Set("userID", int64(userID))
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
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
		userID := c.GetInt64("userID")
		var note model.Note
		if err := c.ShouldBindJSON(&note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный JSON"})
			return
		}

		created, err := noteService.CreateNote(
			c.Request.Context(),
			userID,
			service.CreateNoteInput{
				Title:    note.Title,
				Content:  note.Content,
				Tags:     note.Tags,
				IsPublic: note.IsPublic,
				Priority: note.Priority,
				Category: note.Category,
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
		userID := c.GetInt64("userID")
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		note, err := noteService.GetNote(c.Request.Context(), userID, id)
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
		userID := c.GetInt64("userID")
		notes, err := noteService.ListNotes(c.Request.Context(), userID)
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
		userID := c.GetInt64("userID")
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
			userID,
			id,
			service.UpdateNoteInput{
				Title:    updated.Title,
				Content:  updated.Content,
				Tags:     updated.Tags,
				IsPublic: updated.IsPublic,
				Priority: updated.Priority,
				Category: updated.Category,
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
		userID := c.GetInt64("userID")
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		if err := noteService.DeleteNote(c.Request.Context(), userID, id); err != nil {
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
