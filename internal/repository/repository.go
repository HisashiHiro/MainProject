package repository

import (
	"MainProject/internal/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // драйвер PostgreSQL
	"github.com/redis/go-redis/v9"
)

// Repository — репозиторий для хранения сущностей в PostgreSQL и Redis для аудита
type Repository struct {
	db          *sqlx.DB
	redisClient *redis.Client
	auditTTL    time.Duration

	// Канал для приёма сущностей от планировщика
	inputChan chan []model.Entity
	// Сохранение контекста для отслеживания завершения
	ctx context.Context
	// Указатель на WaitGroup для учёта горутины
	wg *sync.WaitGroup
}

// NewRepository создаёт новый репозиторий
func NewRepository(ctx context.Context, wg *sync.WaitGroup) *Repository {
	repo := &Repository{
		inputChan: make(chan []model.Entity, 100), // Буферизованный канал
		ctx:       ctx,
		wg:        wg,
	}

	if err := repo.initPostgres(ctx); err != nil {
		log.Printf("PostgreSQL  init error: %v", err)
		return nil
	}
	if err := repo.initRedis(ctx); err != nil {
		log.Printf("Redis init error: %v", err)
		return nil
	}

	wg.Add(1)
	go repo.processEntities()

	return repo
}

// Метод для получения канала для возможности сервисом отправлять данные
func (r *Repository) InputChannel() chan<- []model.Entity {
	return r.inputChan
}

// Close закрывает соединения с внешними хранилищами
func (r *Repository) Close(ctx context.Context) {
	if r.db != nil {
		_ = r.db.Close()
	}
	if r.redisClient != nil {
		_ = r.redisClient.Close()
	}
}

// processEntities — горутина, непрерывно обрабатывающая входящие сущности из канала inputChan
func (r *Repository) processEntities() {
	// Откладываем вызов wg.Done() до завершения функции
	defer r.wg.Done()

	for {
		// Одновременное ожидание:
		// 1. Данных из канала inputChan
		// 2. Сигнала отмены из контекста (ctx.Done())
		select {
		// Бесконечный цикл чтения из канала inputChan
		case entities, ok := <-r.inputChan:
			// Если канал закрыт (ok == false), завершаем работу
			if !ok {
				return
			}
			log.Printf("Получено %d сущностей для обработки", len(entities))
			// Обработка каждой сущности в полученной группе
			for _, entity := range entities {
				// Определение конкретного типа сущности через type switch
				// и запись объекта в соответствующую коллекцию MongoDB
				switch v := entity.(type) {
				case *model.Note:
					if _, err := r.CreateNote(r.ctx, v); err != nil {
						log.Printf("Ошибка сохранения заметки: %v", err)
					}
				case *model.User:
					if err := r.insertUser(r.ctx, v); err != nil {
						log.Printf("Ошибка сохранения пользователя: %v", err)
					}
				case *model.Session:
					if err := r.insertSession(r.ctx, v); err != nil {
						log.Printf("Ошибка сохранения сессии: %v", err)
					}
				case *model.Tag:
					if err := r.insertTag(r.ctx, v); err != nil {
						log.Printf("Ошибка сохранения тега: %v", err)
					}
				default:
					log.Printf("Неизвестный тип сущности: %T (значение: %v)", entity, entity)

				}
			}
		case <-r.ctx.Done():
			select {
			case <-r.inputChan: // Пытаемся прочитать, чтобы проверить, закрыт ли канал
				// Канал уже закрыт
			default:
				// Канал ещё открыт
				// Получен сигнал отмены (вызван cancel())
				// Закрытие канала inputChan
				close(r.inputChan)
				log.Println("Канал inputChan закрыт по сигналу контекста")
			}
			return // Завершение горутины
		}
	}
}

// ---- Инициализация PostgreSQL ------------------------------------------------
func (r *Repository) initPostgres(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Значение по умолчанию для локальной разработки
		dsn = "postgres://postgres:postgres@localhost:5432/notes_app?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Проверка соединения
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Опционально: установка лимитов соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	r.db = db
	log.Println("PostgreSQL connected successfully")
	return nil
}

// ---- Инициализация Redis ------------------------------------------------

func (r *Repository) initRedis(ctx context.Context) error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	db := 0 // можно вынести в переменную окружения

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	ttl := 7 * 24 * time.Hour
	if raw := os.Getenv("REDIS_AUDIT_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}

	r.redisClient = client
	r.auditTTL = ttl
	log.Println("Redis connected successfully")

	// Индексы/ограничения можно добавлять здесь***.
	return nil
}

// ---- Вспомогательные функции для аудита в Redis ---------------------------
type auditEvent struct {
	EventID    string          `json:"event_id"`
	EntityType string          `json:"entity_type"`
	EntityID   interface{}     `json:"entity_id"`
	Action     string          `json:"action"`
	OccurredAt time.Time       `json:"occurred_at"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
}

func (r *Repository) writeAuditEvent(ctx context.Context, ev auditEvent) {
	if r.redisClient == nil || r.auditTTL <= 0 {
		return
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		log.Printf("audit marshal error: %v", err)
		return
	}

	eventKey := "audit:event:" + ev.EventID
	indexKey := "audit:index:" + ev.EntityType + ":" + toString(ev.EntityID)

	// Требование "TTL на каждое значение": каждое событие — отдельный ключ с EXPIRE
	if err := r.redisClient.Set(ctx, eventKey, payload, r.auditTTL).Err(); err != nil {
		log.Printf("audit redis SET error: %v", err)
		return
	}

	// Индексируем событие по сущности, чтобы историю можно было собрать по списку
	pipe := r.redisClient.Pipeline()
	pipe.LPush(ctx, indexKey, eventKey)
	pipe.Expire(ctx, indexKey, r.auditTTL)
	_, _ = pipe.Exec(ctx)
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case int64:
		return fmtInt64(x)
	case int:
		return fmtInt64(int64(x))
	case string:
		return x
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func fmtInt64(v int64) string {
	// Без fmt, чтобы не тянуть лишние аллокации в горячем пути
	buf := make([]byte, 0, 20)
	neg := v < 0
	if neg {
		v = -v
	}
	if v == 0 {
		buf = append(buf, '0')
	} else {
		var tmp [20]byte
		i := len(tmp)
		for v > 0 {
			i--
			tmp[i] = byte('0' + v%10)
			v /= 10
		}
		buf = append(buf, tmp[i:]...)
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}

func newEventID() string {
	return fmtInt64(time.Now().UnixNano()) + "-" + fmtInt64(int64(os.Getpid()))
}

func marshalOrNil(v interface{}) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// ---- Реализация методов для работы с заметками (NoteRepository) ------------------

// CreateNote сохраняет заметку в БД (без тегов). Возвращает ID
func (r *Repository) CreateNote(ctx context.Context, note *model.Note) (int64, error) {
	query := `
		INSERT INTO notes (title, content, created_at, updated_at, is_public, is_generated, priority, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		note.Title,
		note.Content,
		note.CreatedAt,
		note.UpdatedAt,
		note.IsPublic,
		note.IsGenerated,
		note.Priority,
		note.Category,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	note.ID = id

	// Аудит
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(note),
	})

	return id, nil
}

// CreateNoteWithTags создаёт заметку вместе с тегами в одной транзакции
// Транзакционный кейс: изменение данных в двух таблицах (notes и note_tags)
func (r *Repository) CreateNoteWithTags(ctx context.Context, note *model.Note, tags []string) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Вставляем заметку
	query := `
		INSERT INTO notes (title, content, created_at, updated_at, is_public, is_generated, priority, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var noteID int64
	err = tx.GetContext(ctx, &noteID, query,
		note.Title,
		note.Content,
		note.CreatedAt,
		note.UpdatedAt,
		note.IsPublic,
		note.IsGenerated,
		note.Priority,
		note.Category,
	)
	if err != nil {
		return 0, err
	}
	note.ID = noteID

	// 2. Обрабатываем теги
	for _, tagName := range tags {
		var tagID int64
		// Пытаемся найти существующий тег
		err = tx.GetContext(ctx, &tagID, `SELECT id FROM tags WHERE name = $1`, tagName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Создаём новый тег
				err = tx.GetContext(ctx, &tagID, `INSERT INTO tags (name, is_generated) VALUES ($1, $2) RETURNING id`, tagName, false)
				if err != nil {
					return 0, err
				}
			} else {
				return 0, err
			}
		}
		// Связываем
		_, err = tx.ExecContext(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)`, noteID, tagID)
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	// Аудит
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   noteID,
		Action:     "create_with_tags",
		OccurredAt: time.Now(),
		After:      marshalOrNil(map[string]interface{}{"note": note, "tags": tags}),
	})

	return noteID, nil
}

// FindNoteByID возвращает заметку по ID, включая теги
func (r *Repository) FindNoteByID(ctx context.Context, id int64) (*model.Note, bool, error) {
	var note model.Note
	query := `SELECT id, title, content, created_at, updated_at, is_public, is_generated, priority, category FROM notes WHERE id = $1`
	err := r.db.GetContext(ctx, &note, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	// Загружаем теги
	tagsQuery := `SELECT t.name FROM tags t JOIN note_tags nt ON t.id = nt.tag_id WHERE nt.note_id = $1 ORDER BY t.name`
	var tags []string
	err = r.db.SelectContext(ctx, &tags, tagsQuery, id)
	if err != nil {
		log.Printf("FindNoteByID error: %v", err)
		return nil, false, err
	}
	note.Tags = tags

	return &note, true, nil
}

// ListNotes возвращает все заметки с тегами
func (r *Repository) ListNotes(ctx context.Context) ([]*model.Note, error) {
	var notes []*model.Note
	query := `SELECT id, title, content, created_at, updated_at, is_public, is_generated, priority, category FROM notes ORDER BY id`
	err := r.db.SelectContext(ctx, &notes, query)
	if err != nil {
		return nil, err
	}

	// Загружаем теги для каждой заметки (можно оптимизировать одним запросом, но для простоты оставим цикл)
	for _, n := range notes {
		tagsQuery := `SELECT t.name FROM tags t JOIN note_tags nt ON t.id = nt.tag_id WHERE nt.note_id = $1 ORDER BY t.name`
		var tags []string
		if err := r.db.SelectContext(ctx, &tags, tagsQuery, n.ID); err != nil {
			return nil, err
		}
		n.Tags = tags
	}
	return notes, nil
}

// UpdateNote обновляет заметку и её теги в транзакции
func (r *Repository) UpdateNote(ctx context.Context, note *model.Note) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// Получаем старые теги для аудита (опционально)
	var oldTags []string
	err = tx.SelectContext(ctx, &oldTags, `SELECT name FROM tags JOIN note_tags ON tags.id = note_tags.tag_id WHERE note_tags.note_id = $1`, note.ID)
	if err != nil {
		return false, err
	}

	// Обновляем заметку
	query := `
		UPDATE notes
		SET title = $1, content = $2, updated_at = $3, is_public = $4, is_generated = $5, priority = $6, category = $7
		WHERE id = $8
	`
	res, err := tx.ExecContext(ctx, query,
		note.Title,
		note.Content,
		note.UpdatedAt,
		note.IsPublic,
		note.IsGenerated,
		note.Priority,
		note.Category,
		note.ID,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}

	// Удаляем старые связи с тегами
	_, err = tx.ExecContext(ctx, `DELETE FROM note_tags WHERE note_id = $1`, note.ID)
	if err != nil {
		return false, err
	}

	// Добавляем новые теги
	for _, tagName := range note.Tags {
		var tagID int64
		// Поиск существующего тега
		err = tx.GetContext(ctx, &tagID, `SELECT id FROM tags WHERE name = $1`, tagName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Создаём новый тег
				err = tx.GetContext(ctx, &tagID, `INSERT INTO tags (name, is_generated) VALUES ($1, $2) RETURNING id`, tagName, false)
				if err != nil {
					return false, err
				}
			} else {
				return false, err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)`, note.ID, tagID)
		if err != nil {
			return false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}

	// Аудит
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   note.ID,
		Action:     "update",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(map[string]interface{}{"tags": oldTags}),
		After:      marshalOrNil(note),
	})

	return true, nil
}

// DeleteNote удаляет заметку и связи с тегами (каскадно в БД, но для аудита нужно)
func (r *Repository) DeleteNote(ctx context.Context, id int64) (bool, error) {
	// Для аудита получим данные до удаления
	var beforeNote model.Note
	err := r.db.GetContext(ctx, &beforeNote, `SELECT * FROM notes WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	// Удаляем (связи удалятся каскадно)
	res, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "delete",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(beforeNote),
	})

	return true, nil
}

// ---- Методы для других сущностей (пользователи, сессии, теги) -----------------

func (r *Repository) insertUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, last_login, is_active, is_generated, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
		user.LastLogin,
		user.IsActive,
		user.IsGenerated,
		user.Role,
	).Scan(&id)
	if err != nil {
		return err
	}
	user.ID = id

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "user",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(user),
	})
	return nil
}

func (r *Repository) insertSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, expires_at, ip, browser, is_generated, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.ExpiresAt,
		session.IP,
		session.Browser,
		session.IsGenerated,
		session.DeviceType,
	)
	if err != nil {
		return err
	}

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "session",
		EntityID:   session.ID,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(session),
	})
	return nil
}

func (r *Repository) insertTag(ctx context.Context, tag *model.Tag) error {
	query := `
		INSERT INTO tags (name, is_generated, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		tag.Tagname,
		tag.IsGenerated,
		tag.Description,
	).Scan(&id)
	if err != nil {
		return err
	}
	tag.ID = id

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "tag",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(tag),
	})
	return nil
}

// Совместимость со старым SchedulerService

func (r *Repository) GetNotes() []model.Entity {
	notes, err := r.ListNotes(r.ctx)
	if err != nil {
		return nil
	}
	out := make([]model.Entity, 0, len(notes))
	for _, n := range notes {
		out = append(out, n)
	}
	return out
}

// Возвращает всех пользователей как слайс Entity
func (r *Repository) GetUsers() []model.Entity {
	var users []*model.User
	query := `SELECT * FROM users ORDER BY id`
	err := r.db.SelectContext(r.ctx, &users, query)
	if err != nil {
		log.Printf("GetUsers error: %v", err)
		return nil
	}
	out := make([]model.Entity, 0, len(users))
	for _, u := range users {
		out = append(out, u)
	}
	return out
}

// Возвращает все сессии как слайс Entity
func (r *Repository) GetSessions() []model.Entity {
	var sessions []*model.Session
	query := `SELECT * FROM sessions`
	err := r.db.SelectContext(r.ctx, &sessions, query)
	if err != nil {
		log.Printf("GetSessions error: %v", err)
		return nil
	}
	out := make([]model.Entity, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, s)
	}
	return out
}

// Возвращает все теги как слайс Entity
func (r *Repository) GetTags() []model.Entity {
	var tags []*model.Tag
	query := `SELECT * FROM tags ORDER BY id`
	err := r.db.SelectContext(r.ctx, &tags, query)
	if err != nil {
		log.Printf("GetTags error: %v", err)
		return nil
	}
	out := make([]model.Entity, 0, len(tags))
	for _, t := range tags {
		out = append(out, t)
	}
	return out
}
