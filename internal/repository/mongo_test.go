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

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	mongoDBName = "testdb"
)

func TestMain(m *testing.M) {
	// Поднимаем контейнер MongoDB
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "6",
		Env:        []string{},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("27017/tcp")
	mongoURI := fmt.Sprintf("mongodb://%s", hostAndPort)

	resource.Expire(120)

	// Ожидаем готовности MongoDB
	pool.MaxWait = 60 * time.Second
	if err = pool.Retry(func() error {
		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
		if err != nil {
			return err
		}
		defer client.Disconnect(context.Background())
		return client.Ping(context.Background(), nil)
	}); err != nil {
		log.Fatalf("Could not connect to MongoDB: %s", err)
	}

	// Создаём постоянный клиент для тестов
	mongoClient, err = mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to create mongo client: %v", err)
	}

	// Запускаем тесты
	code := m.Run()

	// Очистка
	mongoClient.Disconnect(context.Background())
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

// initCollection очищает коллекцию и создаёт необходимые индексы (если нужно)
func initCollection(t *testing.T, coll *mongo.Collection) {
	ctx := context.Background()
	err := coll.Drop(ctx)
	require.NoError(t, err)

	// Если нужны индексы, создаём здесь
	// _, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "user_id", Value: 1}}})
	// require.NoError(t, err)
}

func TestMongoRepository_CreateAndFindNote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx := context.Background()
	db := mongoClient.Database(mongoDBName)
	initCollection(t, db.Collection("notes"))
	initCollection(t, db.Collection("counters"))

	// Создаём репозиторий, передавая nil для Redis (аудит отключён)
	repo := repository.NewMongoRepositoryWithClient(db, nil, 0)

	userID := int64(1)
	expiresAt := time.Now().Add(24 * time.Hour)
	note := model.NewNote("Mongo Title", "Mongo Content", []string{"mongo", "test"}, false, &expiresAt)
	id, err := repo.CreateNote(ctx, note, userID)
	require.NoError(t, err)
	assert.True(t, id > 0)

	found, ok, err := repo.FindNoteByID(ctx, id, userID)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Mongo Title", found.Title)
	assert.Equal(t, userID, found.UserID)
	assert.ElementsMatch(t, []string{"mongo", "test"}, found.Tags)
	assert.NotNil(t, found.ExpiresAt)
	assert.WithinDuration(t, expiresAt, *found.ExpiresAt, time.Second)
}

func TestMongoRepository_ListNotes(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	db := mongoClient.Database(mongoDBName)
	initCollection(t, db.Collection("notes"))
	initCollection(t, db.Collection("counters"))

	repo := repository.NewMongoRepositoryWithClient(db, nil, 0)

	userID := int64(1)
	note1 := model.NewNote("Title1", "Content1", []string{"a"}, false, nil)
	note2 := model.NewNote("Title2", "Content2", []string{"b"}, false, nil)

	id1, err := repo.CreateNote(ctx, note1, userID)
	require.NoError(t, err)
	id2, err := repo.CreateNote(ctx, note2, userID)
	require.NoError(t, err)

	notes, err := repo.ListNotes(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, notes, 2)

	// Проверяем, что для другого пользователя заметок нет
	notesOther, err := repo.ListNotes(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, notesOther)
}

func TestMongoRepository_UpdateNote(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	db := mongoClient.Database(mongoDBName)
	initCollection(t, db.Collection("notes"))
	initCollection(t, db.Collection("counters"))

	repo := repository.NewMongoRepositoryWithClient(db, nil, 0)

	userID := int64(1)
	note := model.NewNote("Old", "Old content", []string{"old"}, false, nil)
	id, err := repo.CreateNote(ctx, note, userID)
	require.NoError(t, err)

	note.ID = id
	note.Title = "Updated"
	note.Content = "Updated content"
	note.Tags = []string{"new", "updated"}
	note.IsPublic = true
	expires := time.Now().Add(48 * time.Hour)
	note.ExpiresAt = &expires

	ok, err := repo.UpdateNote(ctx, note, userID)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, found, err := repo.FindNoteByID(ctx, id, userID)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Updated", updated.Title)
	assert.ElementsMatch(t, []string{"new", "updated"}, updated.Tags)
	assert.True(t, updated.IsPublic)
	assert.NotNil(t, updated.ExpiresAt)
	assert.WithinDuration(t, expires, *updated.ExpiresAt, time.Second)

	// Попытка обновить чужую заметку
	ok, err = repo.UpdateNote(ctx, note, 999)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestMongoRepository_DeleteNote(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	db := mongoClient.Database(mongoDBName)
	initCollection(t, db.Collection("notes"))
	initCollection(t, db.Collection("counters"))

	repo := repository.NewMongoRepositoryWithClient(db, nil, 0)

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

	// Удаление несуществующей заметки
	ok, err = repo.DeleteNote(ctx, 999, userID)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMongoRepository_CreateAndFindUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	db := mongoClient.Database(mongoDBName)
	initCollection(t, db.Collection("users"))
	initCollection(t, db.Collection("counters"))

	repo := repository.NewMongoRepositoryWithClient(db, nil, 0)

	user := model.NewUser("testuser", "test@example.com", []byte("hash"))
	err := repo.CreateUser(ctx, user)
	require.NoError(t, err)
	assert.True(t, user.ID > 0)

	// Поиск по логину
	found, err := repo.GetUserByLogin(ctx, "testuser")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "testuser", found.Username)

	// Поиск по email
	foundByEmail, err := repo.GetUserByLogin(ctx, "test@example.com")
	require.NoError(t, err)
	assert.NotNil(t, foundByEmail)
	assert.Equal(t, "testuser", foundByEmail.Username)

	// Поиск по ID
	foundByID, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.NotNil(t, foundByID)
	assert.Equal(t, "testuser", foundByID.Username)

	// Обновление last login
	err = repo.UpdateLastLogin(ctx, user.ID)
	require.NoError(t, err)

	// Проверка, что last_login обновился
	updated, _ := repo.GetUserByID(ctx, user.ID)
	assert.True(t, updated.LastLogin.After(user.LastLogin) || !updated.LastLogin.IsZero())
}
