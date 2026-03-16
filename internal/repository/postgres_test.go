//go:build integration
// +build integration

package repository_test

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB      *sqlx.DB
	redisClient *redis.Client
)

func TestMain(m *testing.M) {
	// Поднимаем контейнер с PostgreSQL
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=testpass",
			"POSTGRES_USER=testuser",
			"POSTGRES_DB=testdb",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://testuser:testpass@%s/testdb?sslmode=disable", hostAndPort)

	resource.Expire(120) // контейнер удалится через 2 минуты после окончания тестов

	// Ожидаем готовности БД
	pool.MaxWait = 60 * time.Second
	if err = pool.Retry(func() error {
		db, err := sqlx.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Открываем постоянное соединение для тестов
	testDB = sqlx.MustConnect("postgres", databaseUrl)

	// Создаём схему (миграции)
	createTables(testDB)

	// Поднимаем Redis для тестов (можно также через dockertest, но для простоты используем уже запущенный локально)
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Redis not available, audit tests will be skipped: %v", err)
		redisClient = nil
	}

	// Запускаем тесты
	code := m.Run()

	// Очистка
	testDB.Close()
	if redisClient != nil {
		redisClient.Close()
	}
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createTables(db *sqlx.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMPTZ,
		is_public BOOLEAN NOT NULL DEFAULT FALSE,
		is_generated BOOLEAN NOT NULL DEFAULT FALSE,
		priority INT DEFAULT 0,
		category VARCHAR(100)
	);
	CREATE TABLE IF NOT EXISTS tags (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) UNIQUE NOT NULL,
		is_generated BOOLEAN NOT NULL DEFAULT FALSE,
		description TEXT
	);
	CREATE TABLE IF NOT EXISTS note_tags (
		note_id INT REFERENCES notes(id) ON DELETE CASCADE,
		tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (note_id, tag_id)
	);
	`
	db.MustExec(schema)
}

func TestPostgresRepository_CreateAndFindNote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx := context.Background()
	repo := repository.NewPostgresRepositoryWithDB(testDB, redisClient, time.Hour) // нужен конструктор, принимающий *sqlx.DB

	userID := int64(1)
	note := model.NewNote("Integration Title", "Integration Content", []string{"test", "db"}, false, nil)
	id, err := repo.CreateNote(ctx, note, userID)
	require.NoError(t, err)
	assert.True(t, id > 0)

	found, ok, err := repo.FindNoteByID(ctx, id, userID)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Integration Title", found.Title)
	assert.Equal(t, userID, found.UserID)
	assert.ElementsMatch(t, []string{"test", "db"}, found.Tags)
}

func TestPostgresRepository_UpdateNote(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	repo := repository.NewPostgresRepositoryWithDB(testDB, redisClient, time.Hour)

	userID := int64(1)
	note := model.NewNote("Old", "Old content", []string{"old"}, false, nil)
	id, err := repo.CreateNote(ctx, note, userID)
	require.NoError(t, err)

	note.ID = id
	note.Title = "New"
	note.Content = "New content"
	note.Tags = []string{"new", "updated"}
	ok, err := repo.UpdateNote(ctx, note, userID)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, found, err := repo.FindNoteByID(ctx, id, userID)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "New", updated.Title)
	assert.ElementsMatch(t, []string{"new", "updated"}, updated.Tags)
}

func TestPostgresRepository_DeleteNote(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	repo := repository.NewPostgresRepositoryWithDB(testDB, redisClient, time.Hour)

	userID := int64(1)
	note := model.NewNote("ToDelete", "Content", nil, false, nil)
	id, err := repo.CreateNote(ctx, note, userID)
	require.NoError(t, err)

	ok, err := repo.DeleteNote(ctx, id, userID)
	require.NoError(t, err)
	assert.True(t, ok)

	_, found, err := repo.FindNoteByID(ctx, id, userID)
	require.NoError(t, err)
	assert.False(t, found)
}
