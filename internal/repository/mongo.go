package repository

import (
	"MainProject/internal/model"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository реализует NoteRepository для MongoDB
type MongoRepository struct {
	client       *mongo.Client
	db           *mongo.Database
	notesColl    *mongo.Collection
	usersColl    *mongo.Collection
	countersColl *mongo.Collection
	redisClient  *redis.Client
	auditTTL     time.Duration
}

// NewMongoRepository создаёт новый MongoDB репозиторий
func NewMongoRepository(ctx context.Context) *MongoRepository {
	repo := &MongoRepository{}
	if err := repo.initMongo(ctx); err != nil {
		log.Printf("MongoDB init error: %v", err)
		return nil
	}
	if err := repo.initRedis(ctx); err != nil {
		log.Printf("Redis init error: %v", err)
		return nil
	}
	repo.notesColl = repo.db.Collection("notes")
	repo.usersColl = repo.db.Collection("users")
	repo.countersColl = repo.db.Collection("counters")
	repo.initCounter(ctx, "noteid")
	repo.initCounter(ctx, "userid")
	return repo
}

func (r *MongoRepository) initMongo(ctx context.Context) error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "notes_app"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}
	r.client = client
	r.db = client.Database(dbName)
	log.Println("MongoDB connected successfully")
	return nil
}

func (r *MongoRepository) initRedis(ctx context.Context) error {
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

// initCounter создаёт счётчик для автоинкремента ID
func (r *MongoRepository) initCounter(ctx context.Context, counterName string) {
	filter := bson.M{"_id": counterName}
	update := bson.M{"$setOnInsert": bson.M{"seq": 0}}
	opts := options.Update().SetUpsert(true)
	_, err := r.countersColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Failed to init counter %s: %v", counterName, err)
	}
}

// getNextID атомарно увеличивает счётчик
func (r *MongoRepository) getNextID(ctx context.Context, counterName string) (int64, error) {
	filter := bson.M{"_id": counterName}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var result struct {
		Seq int64 `bson:"seq"`
	}
	err := r.countersColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Seq, nil
}

// Close закрывает соединения
func (r *MongoRepository) Close(ctx context.Context) {
	if r.client != nil {
		_ = r.client.Disconnect(ctx)
	}
	if r.redisClient != nil {
		_ = r.redisClient.Close()
	}
}

// ========== Методы для пользователей (UserRepository) ==========

type mongoUser struct {
	ID           int64      `bson:"_id"`
	Username     string     `bson:"username"`
	Email        string     `bson:"email"`
	PasswordHash []byte     `bson:"password_hash"`
	CreatedAt    time.Time  `bson:"created_at"`
	LastLogin    *time.Time `bson:"last_login"`
	IsActive     bool       `bson:"is_active"`
	IsGenerated  bool       `bson:"is_generated"`
	Role         string     `bson:"role"`
}

// CreateUser создаёт нового пользователя
func (r *MongoRepository) CreateUser(ctx context.Context, user *model.User) error {
	id, err := r.getNextID(ctx, "userid")
	if err != nil {
		return err
	}
	mUser := mongoUser{
		ID:           id,
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		LastLogin:    user.LastLogin,
		IsActive:     user.IsActive,
		IsGenerated:  user.IsGenerated,
		Role:         user.Role,
	}
	_, err = r.usersColl.InsertOne(ctx, mUser)
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

// GetUserByLogin возвращает пользователя по логину или email
func (r *MongoRepository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	filter := bson.M{"$or": []bson.M{{"username": login}, {"email": login}}}
	var mUser mongoUser
	err := r.usersColl.FindOne(ctx, filter).Decode(&mUser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &model.User{
		ID:           mUser.ID,
		Username:     mUser.Username,
		Email:        mUser.Email,
		PasswordHash: mUser.PasswordHash,
		CreatedAt:    mUser.CreatedAt,
		LastLogin:    mUser.LastLogin,
		IsActive:     mUser.IsActive,
		IsGenerated:  mUser.IsGenerated,
		Role:         mUser.Role,
	}, nil
}

// GetUserByID возвращает пользователя по ID
func (r *MongoRepository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	filter := bson.M{"_id": id}
	var mUser mongoUser
	err := r.usersColl.FindOne(ctx, filter).Decode(&mUser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &model.User{
		ID:           mUser.ID,
		Username:     mUser.Username,
		Email:        mUser.Email,
		PasswordHash: mUser.PasswordHash,
		CreatedAt:    mUser.CreatedAt,
		LastLogin:    mUser.LastLogin,
		IsActive:     mUser.IsActive,
		IsGenerated:  mUser.IsGenerated,
		Role:         mUser.Role,
	}, nil
}

// UpdateLastLogin обновляет время последнего входа
func (r *MongoRepository) UpdateLastLogin(ctx context.Context, userID int64) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$set": bson.M{"last_login": time.Now()}}
	_, err := r.usersColl.UpdateOne(ctx, filter, update)
	return err
}

// ========== Внутренняя структура для хранения в MongoDB ==========
type mongoNote struct {
	ID          int64      `bson:"_id"`
	UserID      int64      `bson:"user_id"`
	Title       string     `bson:"title"`
	Content     string     `bson:"content"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	ExpiresAt   *time.Time `bson:"expires_at,omitempty"`
	Tags        []string   `bson:"tags"`
	IsPublic    bool       `bson:"is_public"`
	IsGenerated bool       `bson:"is_generated"`
	Priority    int        `bson:"priority"`
	Category    string     `bson:"category"`
}

// ========== Реализация методов NoteRepository ==========

func (r *MongoRepository) CreateNote(ctx context.Context, note *model.Note, userID int64) (int64, error) {
	id, err := r.getNextID(ctx, "noteid")
	if err != nil {
		return 0, err
	}
	mNote := mongoNote{
		ID:          id,
		UserID:      userID,
		Title:       note.Title,
		Content:     note.Content,
		CreatedAt:   note.CreatedAt,
		UpdatedAt:   note.UpdatedAt,
		ExpiresAt:   note.ExpiresAt,
		Tags:        note.Tags,
		IsPublic:    note.IsPublic,
		IsGenerated: note.IsGenerated,
		Priority:    note.Priority,
		Category:    note.Category,
	}
	_, err = r.notesColl.InsertOne(ctx, mNote)
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

func (r *MongoRepository) CreateNoteWithTags(ctx context.Context, note *model.Note, tags []string, userID int64) (int64, error) {
	note.Tags = tags
	return r.CreateNote(ctx, note, userID)
}

func (r *MongoRepository) FindNoteByID(ctx context.Context, id int64, userID int64) (*model.Note, bool, error) {
	var mNote mongoNote
	err := r.notesColl.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&mNote)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	note := &model.Note{
		ID:          mNote.ID,
		UserID:      mNote.UserID,
		Title:       mNote.Title,
		Content:     mNote.Content,
		CreatedAt:   mNote.CreatedAt,
		UpdatedAt:   mNote.UpdatedAt,
		ExpiresAt:   mNote.ExpiresAt,
		Tags:        mNote.Tags,
		IsPublic:    mNote.IsPublic,
		IsGenerated: mNote.IsGenerated,
		Priority:    mNote.Priority,
		Category:    mNote.Category,
	}
	return note, true, nil
}

func (r *MongoRepository) ListNotes(ctx context.Context, userID int64) ([]*model.Note, error) {
	cursor, err := r.notesColl.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notes []*model.Note
	for cursor.Next(ctx) {
		var mNote mongoNote
		if err := cursor.Decode(&mNote); err != nil {
			return nil, err
		}
		notes = append(notes, &model.Note{
			ID:          mNote.ID,
			UserID:      mNote.UserID,
			Title:       mNote.Title,
			Content:     mNote.Content,
			CreatedAt:   mNote.CreatedAt,
			UpdatedAt:   mNote.UpdatedAt,
			ExpiresAt:   mNote.ExpiresAt,
			Tags:        mNote.Tags,
			IsPublic:    mNote.IsPublic,
			IsGenerated: mNote.IsGenerated,
			Priority:    mNote.Priority,
			Category:    mNote.Category,
		})
	}
	return notes, cursor.Err()
}

func (r *MongoRepository) UpdateNote(ctx context.Context, note *model.Note, userID int64) (bool, error) {
	// Проверяем существование и принадлежность
	_, found, err := r.FindNoteByID(ctx, note.ID, userID)
	if err != nil || !found {
		return false, err
	}

	update := bson.M{
		"$set": bson.M{
			"title":        note.Title,
			"content":      note.Content,
			"updated_at":   note.UpdatedAt,
			"expires_at":   note.ExpiresAt,
			"tags":         note.Tags,
			"is_public":    note.IsPublic,
			"is_generated": note.IsGenerated,
			"priority":     note.Priority,
			"category":     note.Category,
		},
	}
	result, err := r.notesColl.UpdateOne(ctx, bson.M{"_id": note.ID, "user_id": userID}, update)
	if err != nil {
		return false, err
	}
	if result.MatchedCount == 0 {
		return false, nil
	}

	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   note.ID,
		Action:     "update",
		OccurredAt: time.Now(),
		After:      marshalOrNil(note),
	})
	return true, nil
}

func (r *MongoRepository) DeleteNote(ctx context.Context, id int64, userID int64) (bool, error) {
	oldNote, found, err := r.FindNoteByID(ctx, id, userID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	result, err := r.notesColl.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return false, err
	}
	if result.DeletedCount == 0 {
		return false, nil
	}
	writeAuditEvent(ctx, r.redisClient, r.auditTTL, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "delete",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(oldNote),
	})
	return true, nil
}

// NewMongoRepositoryWithClient создаёт репозиторий с переданным подключением к БД и Redis (для тестов)
func NewMongoRepositoryWithClient(db *mongo.Database, redisClient *redis.Client, auditTTL time.Duration) *MongoRepository {
	repo := &MongoRepository{
		db:           db,
		redisClient:  redisClient,
		auditTTL:     auditTTL,
		notesColl:    db.Collection("notes"),
		usersColl:    db.Collection("users"),
		countersColl: db.Collection("counters"),
	}
	repo.initCounter(context.Background(), "noteid")
	repo.initCounter(context.Background(), "userid")
	return repo
}
