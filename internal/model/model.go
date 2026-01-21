package model

import "time"

// Entity — общий интерфейс для всех сущностей приложения
type Entity interface {
	GetID() interface{} // Универсальный метод для получения ID любого типа
	EntityType() string // Возвращает тип сущности: "note", "user", "tag", "session"
}

// Note — модель заметки
type Note struct {
	ID        int64     `json:"id"`         // Уникальный идентификатор заметки
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
func (n *Note) GetID() interface{} {
	return n.ID
}

// Устанавливает идентификатор заметки (например, после сохранения в БД)
func (n *Note) SetID(id int64) {
	n.ID = id
}

// Возвращает заголовок заметки
func (n *Note) GetTitle() string {
	return n.Title
}

// Обновляет заголовок
func (n *Note) SetTitle(title string) {
	n.Title = title
	n.UpdatedAt = time.Now()
}

// Возвращает содержимое заметки
func (n *Note) GetContent() string {
	return n.Content
}

// Обновляет содержимое
func (n *Note) SetContent(content string) {
	n.Content = content
	n.UpdatedAt = time.Now()
}

// Возвращает дату создания
func (n *Note) GetCreatedAt() time.Time {
	return n.CreatedAt
}

// Возвращает дату последнего обновления
func (n *Note) GetUpdatedAt() time.Time {
	return n.UpdatedAt
}

// Возвращает список тегов
func (n *Note) GetTags() []string {
	return n.Tags
}

// Добавляет новый тег к заметке и обновляет дату последнего изменения
func (n *Note) AddTag(tag string) {
	n.Tags = append(n.Tags, tag)
	n.UpdatedAt = time.Now()
}

// Возвращает флаг публичности заметки
func (n *Note) GetIsPublic() bool {
	return n.IsPublic
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
	ID           int64     `json:"id"`            // Уникальный идентификатор пользователя
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
func (u *User) GetID() interface{} {
	return u.ID
}

// Устанавливает идентификатор пользователя
func (u *User) SetID(id int64) {
	u.ID = id
}

// Возвращает логин пользователя
func (u *User) GetUsername() string {
	return u.Username
}

// Возвращает email пользователя
func (u *User) GetEmail() string {
	return u.Email
}

// Возвращает хеш пароля
func (u *User) GetPasswordHash() []byte {
	return u.PasswordHash
}

// Обновляет хеш пароля
func (u *User) SetPasswordHash(hash []byte) {
	u.PasswordHash = hash
}

// Возвращает дату регистрации пользователя
func (u *User) GetCreatedAt() time.Time {
	return u.CreatedAt
}

// Возвращает дату последнего входа
func (u *User) GetLastLogin() time.Time {
	return u.LastLogin
}

// Обновляет дату последнего входа
func (u *User) SetLastLogin(t time.Time) {
	u.LastLogin = t
}

// Возвращает статус активности
func (u *User) GetIsActive() bool {
	return u.IsActive
}

// Устанавливает статус активности
func (u *User) SetActive(active bool) {
	u.IsActive = active
}

// Session — модель сессии пользователя
type Session struct {
	ID        string    `json:"id"`         // Уникальный идентификатор сессии (UUID)
	UserID    int64     `json:"user_id"`    // ID пользователя сессии
	ExpiresAt time.Time `json:"expires_at"` // Время истечения сессии
	IP        string    `json:"ip"`         // IP-адрес пользователя
	Browser   string    `json:"browser"`    // Информация о браузере
}

func (u *User) EntityType() string {
	return "user"
}

// ---------------------------------------------------------------
// NewSession создаёт новую сессию с заданными параметрами
func NewSession(id string, userID int64, expiresAt time.Time, ip, browser string) *Session {
	return &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
		IP:        ip,
		Browser:   browser,
	}
}

// Возвращает идентификатор сессии
func (s *Session) GetID() interface{} {
	return s.ID
}

// Возвращает ID пользователя сессии
func (s *Session) GetUserID() int64 {
	return s.UserID
}

// Возвращает время истечения сессии
func (s *Session) GetExpiresAt() time.Time {
	return s.ExpiresAt
}

// Возвращает IP-адрес пользователя
func (s *Session) GetIP() string {
	return s.IP
}

// Возвращает информацию о браузере
func (s *Session) GetBrowser() string {
	return s.Browser
}

// Возвращает тип сущности для интерфейса Entity
func (s *Session) EntityType() string {
	return "session"
}

// ---------------------------------------------------------------
// Tag — модель тега. Приватные поля
// Для поиска заметки по тегу, например: "личные", "рабоота", "покупки"
type Tag struct {
	ID      int64  `json:"id"`      // Уникальный идентификатор тега
	Tagname string `json:"tagname"` // Название тега
}

// NewTag() создаёт новый тег с заданным именем
func NewTag(tagname string) *Tag {
	return &Tag{
		Tagname: tagname,
	}
}

// Возвращает идентификатор тега
func (t *Tag) GetID() interface{} {
	return t.ID
}

// Устанавливает идентификатор тега
func (t *Tag) SetID(id int64) {
	t.ID = id
}

// Возвращает название тега
func (t *Tag) GetTagname() string {
	return t.Tagname
}

// Обновляет название тега
func (t *Tag) SetTagname(tagname string) {
	t.Tagname = tagname
}

// Возвращает тип сущности для интерфейса Entity
func (t *Tag) EntityType() string {
	return "tag"
}
