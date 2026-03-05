package repository

import (
	"MainProject/internal/model"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Repository — репозиторий для хранения сущностей разных типов
type Repository struct {
	mongoClient *mongo.Client
	mongoDB     *mongo.Database

	notesColl    *mongo.Collection
	usersColl    *mongo.Collection
	sessionsColl *mongo.Collection
	tagsColl     *mongo.Collection
	countersColl *mongo.Collection

	redisClient *redis.Client
	auditTTL    time.Duration

	// Канал для приёма сущностей
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

	if err := repo.initMongo(ctx); err != nil {
		log.Printf("MongoDB init error: %v", err)
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
	if r.mongoClient != nil {
		_ = r.mongoClient.Disconnect(ctx)
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

// ---- MongoDB документы и маппинг -------------------------------------------------

type counterDoc struct {
	ID  string `bson:"_id"`
	Seq int64  `bson:"seq"`
}

type noteDoc struct {
	ID          int64     `bson:"_id"`
	Title       string    `bson:"title"`
	Content     string    `bson:"content"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
	Tags        []string  `bson:"tags"`
	IsPublic    bool      `bson:"is_public"`
	IsGenerated bool      `bson:"is_generated"`
}

type userDoc struct {
	ID           int64     `bson:"_id"`
	Username     string    `bson:"username"`
	Email        string    `bson:"email"`
	PasswordHash []byte    `bson:"password_hash"`
	CreatedAt    time.Time `bson:"created_at"`
	LastLogin    time.Time `bson:"last_login"`
	IsActive     bool      `bson:"is_active"`
	IsGenerated  bool      `bson:"is_generated"`
}

type sessionDoc struct {
	ID          string    `bson:"_id"`
	UserID      int64     `bson:"user_id"`
	ExpiresAt   time.Time `bson:"expires_at"`
	IP          string    `bson:"ip"`
	Browser     string    `bson:"browser"`
	IsGenerated bool      `bson:"is_generated"`
}

type tagDoc struct {
	ID          int64  `bson:"_id"`
	Tagname     string `bson:"tagname"`
	IsGenerated bool   `bson:"is_generated"`
}

func noteDocFromModel(id int64, n *model.Note) noteDoc {
	return noteDoc{
		ID:          id,
		Title:       n.Title,
		Content:     n.Content,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
		Tags:        n.Tags,
		IsPublic:    n.IsPublic,
		IsGenerated: n.IsGenerated,
	}
}

func noteModelFromDoc(d noteDoc) *model.Note {
	n := model.NewNote(d.Title, d.Content, d.Tags, d.IsPublic)
	n.SetID(d.ID)
	n.CreatedAt = d.CreatedAt
	n.UpdatedAt = d.UpdatedAt
	n.IsGenerated = d.IsGenerated
	return n
}

func userDocFromModel(id int64, u *model.User) userDoc {
	return userDoc{
		ID:           id,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		LastLogin:    u.LastLogin,
		IsActive:     u.IsActive,
		IsGenerated:  u.IsGenerated,
	}
}

func sessionDocFromModel(id string, s *model.Session) sessionDoc {
	return sessionDoc{
		ID:          id,
		UserID:      s.UserID,
		ExpiresAt:   s.ExpiresAt,
		IP:          s.IP,
		Browser:     s.Browser,
		IsGenerated: s.IsGenerated,
	}
}

func tagDocFromModel(id int64, t *model.Tag) tagDoc {
	return tagDoc{
		ID:          id,
		Tagname:     t.Tagname,
		IsGenerated: t.IsGenerated,
	}
}

// ---- Инициализация MongoDB/Redis ------------------------------------------------

func (r *Repository) initMongo(parent context.Context) error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "notes_app"
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return err
	}

	r.mongoClient = client
	r.mongoDB = client.Database(dbName)

	r.notesColl = r.mongoDB.Collection("notes")
	r.usersColl = r.mongoDB.Collection("users")
	r.sessionsColl = r.mongoDB.Collection("sessions")
	r.tagsColl = r.mongoDB.Collection("tags")
	r.countersColl = r.mongoDB.Collection("counters")

	// Индексы/ограничения можно добавлять здесь, но для учебного проекта достаточно _id.
	return nil
}

func (r *Repository) initRedis(parent context.Context) error {
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

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return err
	}

	ttl := 7 * 24 * time.Hour
	if raw := os.Getenv("REDIS_AUDIT_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}

	r.redisClient = client
	r.auditTTL = ttl
	return nil
}

func (r *Repository) nextSeq(ctx context.Context, name string) (int64, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var out counterDoc
	err := r.countersColl.FindOneAndUpdate(
		ctx,
		bson.M{"_id": name},
		bson.M{"$inc": bson.M{"seq": 1}},
		opts,
	).Decode(&out)
	if err != nil {
		return 0, err
	}
	return out.Seq, nil
}

// Аудит изменений в Redis
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

// Реализация NoteRepository (MongoDB)

func (r *Repository) CreateNote(parent context.Context, note *model.Note) (int64, error) {
	if r.notesColl == nil {
		return 0, errors.New("mongo not initialized")
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	id, err := r.nextSeq(ctx, "notes")
	if err != nil {
		return 0, err
	}

	doc := noteDocFromModel(id, note)
	if _, err := r.notesColl.InsertOne(ctx, doc); err != nil {
		return 0, err
	}

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(doc),
	})

	return id, nil
}

func (r *Repository) FindNoteByID(parent context.Context, id int64) (*model.Note, bool, error) {
	if r.notesColl == nil {
		return nil, false, errors.New("mongo not initialized")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	var doc noteDoc
	err := r.notesColl.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return noteModelFromDoc(doc), true, nil
}

func (r *Repository) ListNotes(parent context.Context) ([]*model.Note, error) {
	if r.notesColl == nil {
		return nil, errors.New("mongo not initialized")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	cur, err := r.notesColl.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make([]*model.Note, 0)
	for cur.Next(ctx) {
		var doc noteDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, noteModelFromDoc(doc))
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) UpdateNote(parent context.Context, note *model.Note) (bool, error) {
	if r.notesColl == nil {
		return false, errors.New("mongo not initialized")
	}

	idAny := note.ID()
	id, ok := idAny.(int64)
	if !ok {
		return false, errors.New("note id must be int64")
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	// Читаем "до" для аудита
	var before noteDoc
	err := r.notesColl.FindOne(ctx, bson.M{"_id": id}).Decode(&before)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	after := noteDocFromModel(id, note)
	res, err := r.notesColl.ReplaceOne(ctx, bson.M{"_id": id}, after)
	if err != nil {
		return false, err
	}
	if res.MatchedCount == 0 {
		return false, nil
	}

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "update",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(before),
		After:      marshalOrNil(after),
	})

	return true, nil
}

func (r *Repository) DeleteNote(parent context.Context, id int64) (bool, error) {
	if r.notesColl == nil {
		return false, errors.New("mongo not initialized")
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	// Читаем "до" для аудита (если нет документа — это не ошибка доменного уровня)
	var before noteDoc
	err := r.notesColl.FindOne(ctx, bson.M{"_id": id}).Decode(&before)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}

	res, err := r.notesColl.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	if res.DeletedCount == 0 {
		return false, nil
	}

	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "note",
		EntityID:   id,
		Action:     "delete",
		OccurredAt: time.Now(),
		Before:     marshalOrNil(before),
	})

	return true, nil
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

func (r *Repository) GetUsers() []model.Entity {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	cur, err := r.usersColl.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	out := make([]model.Entity, 0)
	for cur.Next(ctx) {
		var doc userDoc
		if err := cur.Decode(&doc); err != nil {
			return nil
		}
		u := model.NewUser(doc.Username, doc.Email, doc.PasswordHash)
		u.SetID(doc.ID)
		u.CreatedAt = doc.CreatedAt
		u.LastLogin = doc.LastLogin
		u.IsActive = doc.IsActive
		u.IsGenerated = doc.IsGenerated
		out = append(out, u)
	}
	return out
}

func (r *Repository) GetSessions() []model.Entity {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	cur, err := r.sessionsColl.Find(ctx, bson.D{})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	out := make([]model.Entity, 0)
	for cur.Next(ctx) {
		var doc sessionDoc
		if err := cur.Decode(&doc); err != nil {
			return nil
		}
		s := model.NewSession(doc.ID, doc.UserID, doc.ExpiresAt, doc.IP, doc.Browser)
		s.IsGenerated = doc.IsGenerated
		out = append(out, s)
	}
	return out
}

func (r *Repository) GetTags() []model.Entity {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	cur, err := r.tagsColl.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	out := make([]model.Entity, 0)
	for cur.Next(ctx) {
		var doc tagDoc
		if err := cur.Decode(&doc); err != nil {
			return nil
		}
		t := model.NewTag(doc.Tagname)
		t.SetID(doc.ID)
		t.IsGenerated = doc.IsGenerated
		out = append(out, t)
	}
	return out
}

func (r *Repository) insertUser(parent context.Context, u *model.User) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	idAny := u.ID()
	id, ok := idAny.(int64)
	if !ok || id == 0 {
		newID, err := r.nextSeq(ctx, "users")
		if err != nil {
			return err
		}
		id = newID
		u.SetID(id)
	}
	doc := userDocFromModel(id, u)
	if _, err := r.usersColl.InsertOne(ctx, doc); err != nil {
		return err
	}
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "user",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(doc),
	})
	return nil
}

func (r *Repository) insertTag(parent context.Context, t *model.Tag) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	idAny := t.ID()
	id, ok := idAny.(int64)
	if !ok || id == 0 {
		newID, err := r.nextSeq(ctx, "tags")
		if err != nil {
			return err
		}
		id = newID
		t.SetID(id)
	}
	doc := tagDocFromModel(id, t)
	if _, err := r.tagsColl.InsertOne(ctx, doc); err != nil {
		return err
	}
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "tag",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(doc),
	})
	return nil
}

func (r *Repository) insertSession(parent context.Context, s *model.Session) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	idAny := s.ID()
	id, ok := idAny.(string)
	if !ok || id == "" {
		id = "session-" + fmtInt64(time.Now().UnixNano())
	}
	doc := sessionDocFromModel(id, s)
	if _, err := r.sessionsColl.InsertOne(ctx, doc); err != nil {
		return err
	}
	r.writeAuditEvent(ctx, auditEvent{
		EventID:    newEventID(),
		EntityType: "session",
		EntityID:   id,
		Action:     "create",
		OccurredAt: time.Now(),
		After:      marshalOrNil(doc),
	})
	return nil
}
