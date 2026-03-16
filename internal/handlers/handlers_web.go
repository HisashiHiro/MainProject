package handlers

import (
	"MainProject/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Вспомогательная функция для объединения тегов в строку (используется в шаблонах)
func joinTags(tags []string, sep string) string {
	return strings.Join(tags, sep)
}

// HandleWebLoginPage отображает форму входа
func HandleWebRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

// HandleWebRegister обрабатывает отправку формы регистрации
func HandleWebRegister(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")

		// Простая валидация
		if username == "" || email == "" || password == "" {
			c.HTML(http.StatusBadRequest, "register.html", gin.H{
				"Error":    "Все поля обязательны",
				"Username": username,
				"Email":    email,
			})
			return
		}

		user, err := userService.Register(c.Request.Context(), username, email, password)
		if err != nil {
			errMsg := "Ошибка регистрации"
			if errors.Is(err, service.ErrUserAlreadyExists) {
				errMsg = "Пользователь с таким именем или email уже существует"
			}
			c.HTML(http.StatusBadRequest, "register.html", gin.H{
				"Error":    errMsg,
				"Username": username,
				"Email":    email,
			})
			return
		}

		// Автоматически логиним после регистрации
		session := sessions.Default(c)
		session.Set("userID", user.ID)
		if err := session.Save(); err != nil {
			c.HTML(http.StatusInternalServerError, "register.html", gin.H{
				"Error": "Ошибка при сохранении сессии",
			})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	}
}

// HandleWebLoginPage отображает форму входа
func HandleWebLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// HandleWebLogin обрабатывает отправку формы входа
func HandleWebLogin(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		login := c.PostForm("login")
		password := c.PostForm("password")

		user, err := userService.Login(c.Request.Context(), login, password)
		if err != nil {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{
				"Error": "Неверный логин или пароль",
				"Login": login,
			})
			return
		}

		session.Set("userID", user.ID)
		if err := session.Save(); err != nil {
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{
				"Error": "Ошибка при сохранении сессии",
				"Login": login,
			})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

// HandleWebLogout завершает сессию
func HandleWebLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusSeeOther, "/login")
}

// WebAuthMiddleware проверяет, аутентифицирован ли пользователь
func WebAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("userID")
		if userID == nil {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Set("userID", userID.(int64))
		c.Next()
	}
}

// HandleWebIndex – отображение списка всех заметок
func HandleWebIndex(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		notes, err := noteService.ListNotes(c.Request.Context(), userID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
		c.HTML(http.StatusOK, "index.html", gin.H{"Notes": notes})
	}
}

// HandleWebNewNote – форма создания заметки
func HandleWebNewNote(c *gin.Context) {
	c.HTML(http.StatusOK, "create_note.html", nil)
}

// HandleWebCreateNote – обработка создания заметки
func HandleWebCreateNote(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		title := c.PostForm("title")
		content := c.PostForm("content")
		tagsStr := c.PostForm("tags")
		isPublic := c.PostForm("is_public") == "on"
		expiresAtStr := c.PostForm("expires_at")

		var expiresAt *time.Time
		if expiresAtStr != "" {
			// Ожидаем формат "2006-01-02T15:04" от datetime-local
			t, err := time.Parse("2006-01-02T15:04", expiresAtStr)
			if err == nil {
				expiresAt = &t
			}
			// При ошибке оставляем nil
		}

		// Разбиваем теги
		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}

		_, err := noteService.CreateNote(c.Request.Context(), userID, service.CreateNoteInput{
			Title:     title,
			Content:   content,
			Tags:      tags,
			IsPublic:  isPublic,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			fieldErrors := make(map[string]string)
			if errors.Is(err, service.ErrInvalidNote) {
				if title == "" {
					fieldErrors["title"] = "Заголовок обязателен"
				}
				if content == "" {
					fieldErrors["content"] = "Содержимое обязательно"
				}
			} else {
				fieldErrors["general"] = err.Error()
			}
			// Возвращаем форму с ошибкой и введёнными данными
			c.HTML(http.StatusBadRequest, "create_note.html", gin.H{
				"Title":       title,
				"Content":     content,
				"Tags":        tagsStr,
				"IsPublic":    isPublic,
				"ExpiresAt":   expiresAtStr,
				"FieldErrors": fieldErrors,
				"Error":       "", // общая ошибка не нужна
			})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	}
}

// HandleWebViewNote – просмотр одной заметки
func HandleWebViewNote(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Неверный ID"})
			return
		}
		note, err := noteService.GetNote(c.Request.Context(), userID, id)
		if err != nil {
			if err == service.ErrNoteNotFound {
				c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Заметка не найдена"})
				return
			}
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
		c.HTML(http.StatusOK, "view_note.html", gin.H{
			"Note":      note,
			"TagsAsStr": strings.Join(note.Tags, ", "), // передаём функцию для использования в шаблоне
		})
	}
}

// HandleWebEditNote – форма редактирования заметки
func HandleWebEditNote(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		note, err := noteService.GetNote(c.Request.Context(), userID, id)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Заметка не найдена"})
			return
		}
		expiresAtStr := ""
		if note.ExpiresAt != nil {
			expiresAtStr = note.ExpiresAt.Format("2006-01-02T15:04")
		}
		// Передаём заметку и строку тегов для удобства
		c.HTML(http.StatusOK, "edit_note.html", gin.H{
			"Note":      note,
			"TagsAsStr": strings.Join(note.Tags, ", "),
			"ExpiresAt": expiresAtStr,
		})
	}
}

// HandleWebUpdateNote – обработка обновления заметки
func HandleWebUpdateNote(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Неверный ID"})
			return
		}

		title := c.PostForm("title")
		content := c.PostForm("content")
		tagsStr := c.PostForm("tags")
		isPublic := c.PostForm("is_public") == "on"
		expiresAtStr := c.PostForm("expires_at")

		var expiresAt *time.Time
		if expiresAtStr != "" {
			t, err := time.Parse("2006-01-02T15:04", expiresAtStr)
			if err == nil {
				expiresAt = &t
			}
		}

		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}

		updatedNote, err := noteService.UpdateNote(c.Request.Context(), userID, id, service.UpdateNoteInput{
			Title:     title,
			Content:   content,
			Tags:      tags,
			IsPublic:  isPublic,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			fieldErrors := make(map[string]string)
			if errors.Is(err, service.ErrInvalidNote) {
				if title == "" {
					fieldErrors["title"] = "Заголовок обязателен"
				}
				if content == "" {
					fieldErrors["content"] = "Содержимое обязательно"
				}
			} else if errors.Is(err, service.ErrNoteNotFound) {
				fieldErrors["general"] = "Заметка не найдена"
			} else {
				fieldErrors["general"] = err.Error()
			}
			// В случае ошибки показываем форму редактирования снова
			note, _ := noteService.GetNote(c.Request.Context(), userID, id)
			c.HTML(http.StatusBadRequest, "edit_note.html", gin.H{
				"Note":        note,
				"Title":       title,
				"Content":     content,
				"TagsAsStr":   tagsStr,
				"IsPublic":    isPublic,
				"FieldErrors": fieldErrors,
			})
			return
		}
		c.Redirect(http.StatusSeeOther, "/notes/"+strconv.FormatInt(updatedNote.ID, 10))
	}
}

// HandleWebDeleteNote – удаление заметки
func HandleWebDeleteNote(noteService service.NoteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Неверный ID"})
			return
		}
		if err := noteService.DeleteNote(c.Request.Context(), userID, id); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	}
}
