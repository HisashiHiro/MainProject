package session

import (
	"time"
)

// Session — модель сессии пользователя. Приватные поля
type Session struct {
	id        string    // ID сессии
	userID    int64     // ID пользователя
	expiresAt time.Time // Срок действия сессии
	ip        string    // IP-адрес
	userAgent string    // User-Agent
}

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
func (s *Session) ID() string {
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
