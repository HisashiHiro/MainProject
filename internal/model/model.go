package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Entity — общий интерфейс для всех сущностей приложения
type Entity interface {
	ID() interface{}    // Универсальный метод для получения ID любого типа
	EntityType() string // Возвращает тип сущности: "note", "user", "tag", "session"
}

// Note — модель заметки
type Note struct {
	id        int64     // Уникальный идентификатор заметки
	Title     string    `json:"title"`      // Заголовок заметки
	Content   string    `json:"content"`    // Основное содержимое
	CreatedAt time.Time `json:"created_at"` // Дата/Время создания
	UpdatedAt time.Time `json:"updated_at"` // Дата/Время обновления
	Tags      []string  `json:"tags"`       // Список тегов
	IsPublic  bool      `json:"is_public"`  // Флаг публичности заметки (Нет/Да)
}

// NewNote() создаёт новую заметку с заданными параметрами
// Устанавливает текущие значения createdAt и updatedAt
func NewNote(title, content string, tags []string, isPublic bool) *Note {
	return &Note{
		Title:     title,
		Content:   content,
		Tags:      tags,
		IsPublic:  isPublic,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Возвращает идентификатор заметки
func (n *Note) ID() interface{} {
	return n.id
}

// Реализация интерфейс json.Marshaler
func (n *Note) MarshalJSON() ([]byte, error) {
	var idField interface{}
	switch v := n.ID().(type) {
	case int64:
		idField = v
	case string:
		idField = v
	default:
		return nil, fmt.Errorf("неподдерживаемый тип ID: %T", v)
	}

	type Alias Note
	return json.Marshal(&struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		ID:    idField,
		Alias: (*Alias)(n),
	})
}

// Устанавливает идентификатор заметки (например, после сохранения в БД)
func (n *Note) SetID(id int64) {
	n.id = id
}

// Обновляет заголовок
func (n *Note) SetTitle(title string) {
	n.Title = title
	n.UpdatedAt = time.Now()
}

// Обновляет содержимое
func (n *Note) SetContent(content string) {
	n.Content = content
	n.UpdatedAt = time.Now()
}

// Добавляет новый тег к заметке и обновляет дату последнего изменения
func (n *Note) AddTag(tag string) {
	n.Tags = append(n.Tags, tag)
	n.UpdatedAt = time.Now()
}

// Устанавливает флаг публичности и обновляет дату последнего изменения
func (n *Note) SetPublic(isPublic bool) {
	n.IsPublic = isPublic
	n.UpdatedAt = time.Now()
}

// Возвращает тип сущности для интерфейса Entity
func (n *Note) EntityType() string {
	return "note"
}

// ---------------------------------------------------------------
// User — модель пользователя
type User struct {
	id           int64     // Уникальный идентификатор пользователя
	Username     string    `json:"username"`      // Логин пользователя
	Email        string    `json:"email"`         // Email пользователя
	PasswordHash []byte    `json:"password_hash"` // Хеш пароля
	CreatedAt    time.Time `json:"created_at"`    // Дата регистрации
	LastLogin    time.Time `json:"last_login"`    // Дата последнего входа
	IsActive     bool      `json:"is_active"`     // Статус активности аккаунта
}

// NewUser() создаёт нового пользователя с заданными параметрами
// Устанавливает текущую дату регистрации и активирует аккаунт
func NewUser(username, email string, passwordHash []byte) *User {
	return &User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		IsActive:     true,
	}
}

// Возвращает идентификатор пользователя
func (u *User) ID() interface{} {
	return u.id
}

// Реализация интерфейс json.Marshaler
func (n *User) MarshalJSON() ([]byte, error) {
	var idField interface{}
	switch v := n.ID().(type) {
	case int64:
		idField = v
	case string:
		idField = v
	default:
		return nil, fmt.Errorf("неподдерживаемый тип ID: %T", v)
	}

	type Alias User
	return json.Marshal(&struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		ID:    idField,
		Alias: (*Alias)(n),
	})
}

// Устанавливает идентификатор пользователя
func (u *User) SetID(id int64) {
	u.id = id
}

// Обновляет хеш пароля
func (u *User) SetPasswordHash(hash []byte) {
	u.PasswordHash = hash
}

// Обновляет дату последнего входа
func (u *User) SetLastLogin(t time.Time) {
	u.LastLogin = t
}

// Устанавливает статус активности
func (u *User) SetActive(active bool) {
	u.IsActive = active
}

// Возвращает тип сущности для интерфейса Entity
func (n *User) EntityType() string {
	return "user"
}

// Session — модель сессии пользователя
type Session struct {
	id        string    // Уникальный идентификатор сессии (UUID)
	UserID    int64     `json:"user_id"`    // ID пользователя сессии
	ExpiresAt time.Time `json:"expires_at"` // Время истечения сессии
	IP        string    `json:"ip"`         // IP-адрес пользователя
	Browser   string    `json:"browser"`    // Информация о браузере
}

// ---------------------------------------------------------------
// NewSession создаёт новую сессию с заданными параметрами
func NewSession(id string, userID int64, expiresAt time.Time, ip, browser string) *Session {
	return &Session{
		id:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
		IP:        ip,
		Browser:   browser,
	}
}

// Возвращает идентификатор сессии
func (s *Session) ID() interface{} {
	return s.id
}

// Реализация интерфейс json.Marshaler
func (n *Session) MarshalJSON() ([]byte, error) {
	var idField interface{}
	switch v := n.ID().(type) {
	case int64:
		idField = v
	case string:
		idField = v
	default:
		return nil, fmt.Errorf("неподдерживаемый тип ID: %T", v)
	}

	type Alias Session
	return json.Marshal(&struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		ID:    idField,
		Alias: (*Alias)(n),
	})
}

// Возвращает тип сущности для интерфейса Entity
func (s *Session) EntityType() string {
	return "session"
}

// ---------------------------------------------------------------
// Tag — модель тега. Приватные поля
// Для поиска заметки по тегу, например: "личные", "рабоота", "покупки"
type Tag struct {
	id      int64  // Уникальный идентификатор тега
	Tagname string `json:"tagname"` // Название тега
}

// NewTag() создаёт новый тег с заданным именем
func NewTag(tagname string) *Tag {
	return &Tag{
		Tagname: tagname,
	}
}

// Возвращает идентификатор тега
func (t *Tag) ID() interface{} {
	return t.id
}

// Реализация интерфейс json.Marshaler
func (n *Tag) MarshalJSON() ([]byte, error) {
	var idField interface{}
	switch v := n.ID().(type) {
	case int64:
		idField = v
	case string:
		idField = v
	default:
		return nil, fmt.Errorf("неподдерживаемый тип ID: %T", v)
	}

	type Alias Tag
	return json.Marshal(&struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		ID:    idField,
		Alias: (*Alias)(n),
	})
}

// Устанавливает идентификатор тега
func (t *Tag) SetID(id int64) {
	t.id = id
}

// Обновляет название тега
func (t *Tag) SetTagname(tagname string) {
	t.Tagname = tagname
}

// Возвращает тип сущности для интерфейса Entity
func (t *Tag) EntityType() string {
	return "tag"
}
