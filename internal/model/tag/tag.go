package tag

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
func (t *Tag) ID() int64 {
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
