package model

import "time"

// Entity — общий интерфейс для всех сущностей приложения.
type Entity interface {
	EntityType() string // Возвращает тип сущности: "note", "user", "tag", "session"
	ID() interface{}    // Универсальный метод для получения ID любого типа
}

// Note — модель заметки. Приватные поля
type Note struct {
	id        int64     // ID заметки
	title     string    // Заголовок
	content   string    // Содержимое
	createdAt time.Time // Дата/Время создания
	updatedAt time.Time // Дата/Время обновления
	tags      []string  // Теги
	isPublic  bool      // Публичная заметка (Нет/Да)
}

// NewNote создаёт новую заметку
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

// ID возвращает ID заметки
func (n *Note) ID() interface{} {
	return n.id
}

// SetID устанавливает ID заметки (например, после сохранения в БД)
func (n *Note) SetID(id int64) {
	n.id = id
}

// Title возвращает заголовок заметки
func (n *Note) Title() string {
	return n.title
}

// SetTitle обновляет заголовок
func (n *Note) SetTitle(title string) {
	n.title = title
	n.updatedAt = time.Now()
}

// Content возвращает содержимое заметки
func (n *Note) Content() string {
	return n.content
}

// SetContent обновляет содержимое
func (n *Note) SetContent(content string) {
	n.content = content
	n.updatedAt = time.Now()
}

// CreatedAt возвращает дату создания
func (n *Note) CreatedAt() time.Time {
	return n.createdAt
}

// UpdatedAt возвращает дату последнего обновления
func (n *Note) UpdatedAt() time.Time {
	return n.updatedAt
}

// Tags возвращает список тегов
func (n *Note) Tags() []string {
	return n.tags
}

// AddTag добавляет тег
func (n *Note) AddTag(tag string) {
	n.tags = append(n.tags, tag)
	n.updatedAt = time.Now()
}

// IsPublic возвращает флаг публичности
func (n *Note) IsPublic() bool {
	return n.isPublic
}

// SetPublic устанавливает флаг публичности
func (n *Note) SetPublic(isPublic bool) {
	n.isPublic = isPublic
	n.updatedAt = time.Now()
}

func (n *Note) EntityType() string {
	return "note"
}

// ---------------------------------------------------------------
// User — модель пользователя. Приватные поля
type User struct {
	id           int64     // ID пользователя
	username     string    // Логин
	email        string    // Email
	passwordHash []byte    // Хеш пароля
	createdAt    time.Time // Дата регистрации
	lastLogin    time.Time // Последний вход
	isActive     bool      // Активен ли аккаунт
}

// NewUser создаёт нового пользователя.
func NewUser(username, email string, passwordHash []byte) *User {
	return &User{
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    time.Now(),
		isActive:     true,
	}
}

// ID возвращает ID пользователя.
func (u *User) ID() interface{} {
	return u.id
}

// SetID устанавливает ID пользователя.
func (u *User) SetID(id int64) {
	u.id = id
}

// Username возвращает логин.
func (u *User) Username() string {
	return u.username
}

// Email возвращает email.
func (u *User) Email() string {
	return u.email
}

// PasswordHash возвращает хеш пароля.
func (u *User) PasswordHash() []byte {
	return u.passwordHash
}

// SetPasswordHash обновляет хеш пароля.
func (u *User) SetPasswordHash(hash []byte) {
	u.passwordHash = hash
}

// CreatedAt возвращает дату регистрации.
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// LastLogin возвращает дату последнего входа.
func (u *User) LastLogin() time.Time {
	return u.lastLogin
}

// SetLastLogin обновляет дату последнего входа.
func (u *User) SetLastLogin(t time.Time) {
	u.lastLogin = t
}

// IsActive возвращает статус активности.
func (u *User) IsActive() bool {
	return u.isActive
}

// SetActive устанавливает статус активности.
func (u *User) SetActive(active bool) {
	u.isActive = active
}

// Session — модель сессии пользователя. Приватные поля
type Session struct {
	id        string    // ID сессии
	userID    int64     // ID пользователя
	expiresAt time.Time // Срок действия сессии
	ip        string    // IP-адрес
	userAgent string    // User-Agent
}

func (u *User) EntityType() string {
	return "user"
}

// ---------------------------------------------------------------
// NewSession создаёт новую сессию.
func NewSession(id string, userID int64, expiresAt time.Time, ip, userAgent string) *Session {
	return &Session{
		id:        id,
		userID:    userID,
		expiresAt: expiresAt,
		ip:        ip,
		userAgent: userAgent,
	}
}

// ID возвращает ID сессии.
func (s *Session) ID() interface{} {
	return s.id
}

// UserID возвращает ID пользователя.
func (s *Session) UserID() int64 {
	return s.userID
}

// ExpiresAt возвращает срок действия сессии.
func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

// IP возвращает IP-адрес.
func (s *Session) IP() string {
	return s.ip
}

// UserAgent возвращает User-Agent.
func (s *Session) User()

func (s *Session) EntityType() string {
	return "session"
}

// ---------------------------------------------------------------
// Tag — модель тега. Приватные поля
// Для поиска заметки по тегу, например: "личные", "рабоота", "покупки"
type Tag struct {
	id   int64  // ID тега
	name string // Название тега
}

// NewTag создаёт новый тег.
func NewTag(name string) *Tag {
	return &Tag{
		name: name,
	}
}

// ID возвращает ID тега.
func (t *Tag) ID() interface{} {
	return t.id
}

// SetID устанавливает ID тега.
func (t *Tag) SetID(id int64) {
	t.id = id
}

// Name возвращает название тега.
func (t *Tag) Name() string {
	return t.name
}

// SetName обновляет название тега.
func (t *Tag) SetName(name string) {
	t.name = name
}

func (t *Tag) EntityType() string {
	return "tag"
}
