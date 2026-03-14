package repository

import (
	"MainProject/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// PostgresRepository реализует NoteRepository для PostgreSQL
type PostgresRepository struct {
	db          *sqlx.DB
	redisClient *redis.Client
	auditTTL    time.Duration
}

// NewPostgresRepository создаёт новый PostgreSQL репозиторий
func NewPostgresRepository(ctx context.Context) *PostgresRepository {
	repo := &PostgresRepository{}
	if err := repo.initPostgres(ctx); err != nil {
		log.Printf("PostgreSQL init error: %v", err)
		return nil
	}
	if err := repo.initRedis(ctx); err != nil {
		log.Printf("Redis init error: %v", err)
		return nil
	}
	return repo
}

func (r *PostgresRepository) initPostgres(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/notes_app?sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	r.db = db
	log.Println("PostgreSQL connected successfully")
	return nil
}

func (r *PostgresRepository) initRedis(ctx context.Context) error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	db := 0
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
	return nil
}

// Close закрывает соединения
func (r *PostgresRepository) Close(ctx context.Context) {
	if r.db != nil {
		_ = r.db.Close()
	}
	if r.redisClient != nil {
		_ = r.redisClient.Close()
	}
}

// ========== Методы для пользователей (UserRepository) ==========

// CreateUser создаёт нового пользователя
func (r *PostgresRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, is_active, role, is_generated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.CreatedAt,
		user.IsActive, user.Role, user.IsGenerated,
	).Scan(&user.ID)
	return err
}

// GetUserByLogin возвращает пользователя по логину (username) или email
func (r *PostgresRepository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, email, password_hash, created_at, last_login, is_active, role, is_generated
              FROM users WHERE username = $1 OR email = $1`
	log.Printf("[DEBUG] Executing query: %s with login: %s", query, login)
	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[DEBUG] No user found with login: %s", login)
			return nil, nil
		}
		log.Printf("[ERROR] Database error in GetUserByLogin: %v", err)
		return nil, err
	}
	log.Printf("[DEBUG] User retrieved: ID=%d, username=%s, email=%s", user.ID, user.Username, user.Email)
	return &user, nil
}

/*
func (r *PostgresRepository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, email, password_hash, created_at, last_login, is_active, role, is_generated
	          FROM users WHERE username = $1 OR email = $1`
	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // пользователь не найден
		}
		return nil, err
	}
	return &user, nil
}
*/
// GetUserByID возвращает пользователя по ID
func (r *PostgresRepository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, email, password_hash, created_at, last_login, is_active, role, is_generated
	          FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateLastLogin обновляет время последнего входа пользователя
func (r *PostgresRepository) UpdateLastLogin(ctx context.Context, userID int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login = $1 WHERE id = $2`, &now, userID)
	return err
}

// ========== Реализация методов NoteRepository ==========

func (r *PostgresRepository) CreateNote(ctx context.Context, note *model.Note, userID int64) (int64, error) {
	query := `
		INSERT INTO notes (user_id, title, content, created_at, updated_at, expires_at, is_public, is_generated, priority, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		userID,
		note.Title, note.Content, note.CreatedAt, note.UpdatedAt,
		note.ExpiresAt,
		note.IsPublic, note.IsGenerated, note.Priority, note.Category,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	note.ID = id
	note.UserID = userID
	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(note),
	})
	return id, nil
}

func (r *PostgresRepository) CreateNoteWithTags(ctx context.Context, note *model.Note, tags []string, userID int64) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO notes (user_id, title, content, created_at, updated_at, is_public, is_generated, priority, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var noteID int64
	err = tx.GetContext(ctx, &noteID, query,
		userID,
		note.Title, note.Content, note.CreatedAt, note.UpdatedAt,
		note.IsPublic, note.IsGenerated, note.Priority, note.Category,
	)
	if err != nil {
		return 0, err
	}
	note.ID = noteID
	note.UserID = userID

	for _, tagName := range tags {
		var tagID int64
		err = tx.GetContext(ctx, &tagID, `SELECT id FROM tags WHERE name = $1`, tagName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = tx.GetContext(ctx, &tagID, `INSERT INTO tags (name, is_generated) VALUES ($1, $2) RETURNING id`, tagName, false)
				if err != nil {
					return 0, err
				}
			} else {
				return 0, err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)`, noteID, tagID)
		if err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   noteID,
		Action:     "create_with_tags",
		OccurredAt: time.Now(),
		After:      marshalOrNil(map[string]interface{}{"note": note, "tags": tags}),
	})
	return noteID, nil
}

func (r *PostgresRepository) FindNoteByID(ctx context.Context, id int64, userID int64) (*model.Note, bool, error) {
	var note model.Note
	query := `SELECT id, user_id, title, content, created_at, updated_at, expires_at, is_public, is_generated, priority, category FROM notes WHERE id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &note, query, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
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

func (r *PostgresRepository) ListNotes(ctx context.Context, userID int64) ([]*model.Note, error) {
	var notes []*model.Note
	query := `SELECT id, user_id, title, content, created_at, updated_at, expires_at, is_public, is_generated, priority, category FROM notes WHERE user_id = $1 ORDER BY id`
	err := r.db.SelectContext(ctx, &notes, query, userID)
	if err != nil {
		return nil, err
	}
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

func (r *PostgresRepository) UpdateNote(ctx context.Context, note *model.Note, userID int64) (bool, error) {
	// Проверка на принадлежность заметки пользователю
	_, found, err := r.FindNoteByID(ctx, note.ID, userID)
	if err != nil || !found {
		return false, err
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var oldTags []string
	err = tx.SelectContext(ctx, &oldTags,
		`SELECT name FROM tags JOIN note_tags ON tags.id = note_tags.tag_id WHERE note_tags.note_id = $1`,
		note.ID,
	)
	if err != nil {
		return false, err
	}

	query := `
		UPDATE notes
		SET title = $1, content = $2, updated_at = $3, expires_at = $4,
		    is_public = $5, is_generated = $6, priority = $7, category = $8
		WHERE id = $9 AND user_id = $10
	`
	res, err := tx.ExecContext(ctx, query,
		note.Title, note.Content, note.UpdatedAt, note.ExpiresAt,
		note.IsPublic, note.IsGenerated, note.Priority, note.Category,
		note.ID, userID,
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

	_, err = tx.ExecContext(ctx, `DELETE FROM note_tags WHERE note_id = $1`, note.ID)
	if err != nil {
		return false, err
	}

	for _, tagName := range note.Tags {
		var tagID int64
		err = tx.GetContext(ctx, &tagID, `SELECT id FROM tags WHERE name = $1`, tagName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = tx.GetContext(ctx, &tagID,
					`INSERT INTO tags (name, is_generated) VALUES ($1, $2) RETURNING id`,
					tagName, false,
				)
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
	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
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

func (r *PostgresRepository) DeleteNote(ctx context.Context, id int64, userID int64) (bool, error) {
	var beforeNote model.Note
	err := r.db.GetContext(ctx, &beforeNote, `SELECT * FROM notes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, id, userID)
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
	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "delete",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(beforeNote),
	})
	return true, nil
}

func NewPostgresRepositoryWithDB(db *sqlx.DB, redisClient *redis.Client, auditTTL time.Duration) *PostgresRepository {
	return &PostgresRepository{
		db:          db,
		redisClient: redisClient,
		auditTTL:    auditTTL,
	}
}
