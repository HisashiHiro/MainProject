package model

import "time"

// Entity — общий интерфейс для всех сущностей приложения
type Entity interface {
	ID() interface{}    // Универсальный метод для получения ID любого типа
	EntityType() string // Возвращает тип сущности: "note", "user", "tag", "session"
}

// Note — модель заметки
type Note struct {
	id        int64     // Уникальный идентификатор заметки
	title     string    // Заголовок заметки
	content   string    // Основное содержимое
	createdAt time.Time // Дата/Время создания
	updatedAt time.Time // Дата/Время обновления
	tags      []string  // Список тегов
	isPublic  bool      // Флаг публичности заметки (Нет/Да)
}

// NewNote() создаёт новую заметку с заданными параметрами
// Устанавливает текущие значения createdAt и updatedAt
func NewNote(title, content string, tags []string, isPublic bool) *Note {
	return &Note{
		title:     title,
		content:   content,
		tags:      tags,
		isPublic:  isPublic,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// ID() возвращает идентификатор заметки
func (n *Note) ID() interface{} {
	return n.id
}

// SetID() устанавливает идентификатор заметки (например, после сохранения в БД)
func (n *Note) SetID(id int64) {
	n.id = id
}

// Title() возвращает заголовок заметки
func (n *Note) Title() string {
	return n.title
}

// SetTitle() обновляет заголовок
func (n *Note) SetTitle(title string) {
	n.title = title
	n.updatedAt = time.Now()
}

// Content() возвращает содержимое заметки
func (n *Note) Content() string {
	return n.content
}

// SetContent() обновляет содержимое
func (n *Note) SetContent(content string) {
	n.content = content
	n.updatedAt = time.Now()
}

// CreatedAt() возвращает дату создания
func (n *Note) CreatedAt() time.Time {
	return n.createdAt
}

// UpdatedAt() возвращает дату последнего обновления
func (n *Note) UpdatedAt() time.Time {
	return n.updatedAt
}

// Tags() возвращает список тегов
func (n *Note) Tags() []string {
	return n.tags
}

// AddTag() добавляет новый тег к заметке и обновляет дату последнего изменения
func (n *Note) AddTag(tag string) {
	n.tags = append(n.tags, tag)
	n.updatedAt = time.Now()
}

// IsPublic() возвращает флаг публичности заметки
func (n *Note) IsPublic() bool {
	return n.isPublic
}

// SetPublic() устанавливает флаг публичности и обновляет дату последнего изменения
func (n *Note) SetPublic(isPublic bool) {
	n.isPublic = isPublic
	n.updatedAt = time.Now()
}

// EntityType возвращает тип сущности для интерфейса Entity
func (n *Note) EntityType() string {
	return "note"
}

// ---------------------------------------------------------------
// User — модель пользователя
type User struct {
	id           int64     // Уникальный идентификатор пользователя
	username     string    // Логин пользователя
	email        string    // Email пользователя
	passwordHash []byte    // Хеш пароля
	createdAt    time.Time // Дата регистрации
	lastLogin    time.Time // Дата последнего входа
	isActive     bool      // Статус активности аккаунта
}

// NewUser() создаёт нового пользователя с заданными параметрами
// Устанавливает текущую дату регистрации и активирует аккаунт
func NewUser(username, email string, passwordHash []byte) *User {
	return &User{
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    time.Now(),
		isActive:     true,
	}
}

// ID() возвращает идентификатор пользователя
func (u *User) ID() interface{} {
	return u.id
}

// SetID)() устанавливает идентификатор пользователя
func (u *User) SetID(id int64) {
	u.id = id
}

// Username() возвращает логин пользователя
func (u *User) Username() string {
	return u.username
}

// Email() возвращает email пользователя
func (u *User) Email() string {
	return u.email
}

// PasswordHash() возвращает хеш пароля
func (u *User) PasswordHash() []byte {
	return u.passwordHash
}

// SetPasswordHash() обновляет хеш пароля
func (u *User) SetPasswordHash(hash []byte) {
	u.passwordHash = hash
}

// CreatedAt() возвращает дату регистрации пользователя
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// LastLogin() возвращает дату последнего входа
func (u *User) LastLogin() time.Time {
	return u.lastLogin
}

// SetLastLogin() обновляет дату последнего входа
func (u *User) SetLastLogin(t time.Time) {
	u.lastLogin = t
}

// IsActive() возвращает статус активности
func (u *User) IsActive() bool {
	return u.isActive
}

// SetActive() устанавливает статус активности
func (u *User) SetActive(active bool) {
	u.isActive = active
}

// Session — модель сессии пользователя
type Session struct {
	id        string    // Уникальный идентификатор сессии (UUID)
	userID    int64     // ID пользователя сессии
	expiresAt time.Time // Время истечения сессии
	ip        string    // IP-адрес пользователя
	browser   string    // Информация о браузере
}

func (u *User) EntityType() string {
	return "user"
}

// ---------------------------------------------------------------
// NewSession создаёт новую сессию с заданными параметрами
func NewSession(id string, userID int64, expiresAt time.Time, ip, browser string) *Session {
	return &Session{
		id:        id,
		userID:    userID,
		expiresAt: expiresAt,
		ip:        ip,
		browser:   browser,
	}
}

// ID() возвращает идентификатор сессии
func (s *Session) ID() interface{} {
	return s.id
}

// UserID() возвращает ID пользователя сессии
func (s *Session) UserID() int64 {
	return s.userID
}

// ExpiresAt() возвращает время истечения сессии
func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

// IP() возвращает IP-адрес пользователя
func (s *Session) IP() string {
	return s.ip
}

// Browser() возвращает информацию о браузере
func (s *Session) Browser() string {
	return s.browser
}

// EntityType() возвращает тип сущности для интерфейса Entity
func (s *Session) EntityType() string {
	return "session"
}

// ---------------------------------------------------------------
// Tag — модель тега. Приватные поля
// Для поиска заметки по тегу, например: "личные", "рабоота", "покупки"
type Tag struct {
	id   int64  // Уникальный идентификатор тега
	name string // Название тега
}

// NewTag() создаёт новый тег с заданным именем
func NewTag(name string) *Tag {
	return &Tag{
		name: name,
	}
}

// ID() возвращает идентификатор тега
func (t *Tag) ID() interface{} {
	return t.id
}

// SetID() устанавливает идентификатор тега
func (t *Tag) SetID(id int64) {
	t.id = id
}

// Name() возвращает название тега
func (t *Tag) Name() string {
	return t.name
}

// SetName() обновляет название тега
func (t *Tag) SetName(name string) {
	t.name = name
}

// EntityType() возвращает тип сущности для интерфейса Entity
func (t *Tag) EntityType() string {
	return "tag"
}
