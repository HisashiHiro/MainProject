package note

import (
	"time"
)

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
func (n *Note) ID() int64 {
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
