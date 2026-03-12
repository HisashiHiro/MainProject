package model

import (
	"time"
)

// Entity — общий интерфейс для всех сущностей приложения
type Entity interface {
	EntityType() string // Возвращает тип сущности: "note", "user", "tag", "session"
}

// Note — модель заметки
type Note struct {
	ID          int64     `db:"id" json:"id"`                     // Уникальный идентификатор заметки
	Title       string    `db:"title" json:"title"`               // Заголовок заметки
	Content     string    `db:"content" json:"content"`           // Основное содержимое
	CreatedAt   time.Time `db:"created_at" json:"created_at"`     // Дата/Время создания
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`     // Дата/Время обновления
	Tags        []string  `db:"-" json:"tags"`                    // Список тегов
	IsPublic    bool      `db:"is_public" json:"is_public"`       // Флаг публичности заметки (Нет/Да)
	IsGenerated bool      `db:"is_generated" json:"is_generated"` // Флаг: создана ли сущность сервисом
	Priority    int       `db:"priority" json:"priority"`         // Приоритет заметки
	Category    string    `db:"category" json:"category"`         // Категория
}

// NewNote() создаёт новую заметку с заданными параметрами
// Устанавливает текущие значения createdAt и updatedAt
func NewNote(title, content string, tags []string, isPublic bool) *Note {
	return &Note{
		Title:       title,
		Content:     content,
		Tags:        tags,
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsGenerated: true, // Создана сервисом
		Priority:    0,    // по умолчанию
		Category:    "",   // по умолчанию пусто
	}
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

// Устанавливает приоритет и обновляет время
func (n *Note) SetPriority(priority int) {
	n.Priority = priority
	n.UpdatedAt = time.Now()
}

// Устанавливает категорию и обновляет время
func (n *Note) SetCategory(category string) {
	n.Category = category
	n.UpdatedAt = time.Now()
}

// Возвращает тип сущности для интерфейса Entity
func (n *Note) EntityType() string {
	return "note"
}

// ---------------------------------------------------------------
// User — модель пользователя
type User struct {
	ID           int64     `db:"id" json:"id"`                       // Уникальный идентификатор пользователя
	Username     string    `db:"username" json:"username"`           // Логин пользователя
	Email        string    `db:"email" json:"email"`                 // Email пользователя
	PasswordHash []byte    `db:"password_hash" json:"password_hash"` // Хеш пароля
	CreatedAt    time.Time `db:"created_at" json:"created_at"`       // Дата регистрации
	LastLogin    time.Time `db:"last_login" json:"last_login"`       // Дата последнего входа
	IsActive     bool      `db:"is_active" json:"is_active"`         // Статус активности аккаунта
	IsGenerated  bool      `db:"is_generated" json:"is_generated"`   // Флаг: создана ли сущность сервисом
	Role         string    `db:"role" json:"role"`                   // Роль пользователя
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
		IsGenerated:  true, // Создана сервисом
		Role:         "user",
	}
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

// Устанавливает роль пользователя
func (u *User) SetRole(role string) {
	u.Role = role
}

// Возвращает тип сущности для интерфейса Entity
func (n *User) EntityType() string {
	return "user"
}

// Session — модель сессии пользователя
type Session struct {
	ID          string    `db:"id" json:"id"`                     // Уникальный идентификатор сессии (UUID)
	UserID      int64     `db:"user_id" json:"user_id"`           // ID пользователя сессии
	ExpiresAt   time.Time `db:"expires_at" json:"expires_at"`     // Время истечения сессии
	IP          string    `db:"ip" json:"ip"`                     // IP-адрес пользователя
	Browser     string    `db:"browser" json:"browser"`           // Информация о браузере
	IsGenerated bool      `db:"is_generated" json:"is_generated"` // Флаг: создана ли сущность сервисом
	DeviceType  string    `db:"device_type" json:"device_type"`   // Тип устройства
}

// ---------------------------------------------------------------
// NewSession создаёт новую сессию с заданными параметрами
func NewSession(id string, userID int64, expiresAt time.Time, ip, browser string) *Session {
	return &Session{
		ID:          id,
		UserID:      userID,
		ExpiresAt:   expiresAt,
		IP:          ip,
		Browser:     browser,
		IsGenerated: true, // Создана сервисом
		DeviceType:  "unknown",
	}
}

// Устанавливает время истечения
func (s *Session) SetExpiresAt(t time.Time) {
	s.ExpiresAt = t
}

// Устанавливает IP-адрес
func (s *Session) SetIP(ip string) {
	s.IP = ip
}

// Устанавливает информацию о браузере
func (s *Session) SetBrowser(browser string) {
	s.Browser = browser
}

// Устанавливает тип устройства
func (s *Session) SetDeviceType(deviceType string) {
	s.DeviceType = deviceType
}

// Возвращает тип сущности для интерфейса Entity
func (s *Session) EntityType() string {
	return "session"
}

// ---------------------------------------------------------------
// Tag — модель тега. Приватные поля
// Для поиска заметки по тегу, например: "личные", "рабоота", "покупки"
type Tag struct {
	ID          int64  `db:"id" json:"id"`                     // Уникальный идентификатор тега
	Tagname     string `db:"tagname" json:"tagname"`           // Название тега
	IsGenerated bool   `db:"is_generated" json:"is_generated"` // Флаг: создана ли сущность сервисом
	Description string `db:"description" json:"description"`   // Описание тега
}

// NewTag() создаёт новый тег с заданным именем
func NewTag(tagname string) *Tag {
	return &Tag{
		Tagname:     tagname,
		IsGenerated: true, // Создана сервисом
		Description: "",
	}
}

// Обновляет название тега
func (t *Tag) SetTagname(tagname string) {
	t.Tagname = tagname
}

// Устанавливает описание тега
func (t *Tag) SetDescription(desc string) {
	t.Description = desc
}

// Возвращает тип сущности для интерфейса Entity
func (t *Tag) EntityType() string {
	return "tag"
}
