package user

import (
	"time"
)

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
func (u *User) ID() int64 {
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
