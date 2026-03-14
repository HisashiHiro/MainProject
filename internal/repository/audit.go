package repository

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// auditEvent — структура для событий аудита
type auditEvent struct {
	EventID    string          `json:"event_id"`
	EntityType string          `json:"entity_type"`
	EntityID   interface{}     `json:"entity_id"`
	Action     string          `json:"action"`
	OccurredAt time.Time       `json:"occurred_at"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
}

// writeAuditEvent записывает событие аудита в Redis
func writeAuditEvent(ctx context.Context, redisClient *redis.Client, ttl time.Duration, ev auditEvent) {
	if redisClient == nil || ttl <= 0 {
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
	if err := redisClient.Set(ctx, eventKey, payload, ttl).Err(); err != nil {
		log.Printf("audit redis SET error: %v", err)
		return
	}

	// Индексируем событие по сущности, чтобы историю можно было собрать по списку
	pipe := redisClient.Pipeline()
	pipe.LPush(ctx, indexKey, eventKey)
	pipe.Expire(ctx, indexKey, ttl)
	_, _ = pipe.Exec(ctx)
}

// toString преобразует ID в строку для ключа Redis
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

// fmtInt64 преобразует int64 в строку без fmt
func fmtInt64(v int64) string {
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

// newEventID генерирует уникальный ID события
func newEventID() string {
	return fmtInt64(time.Now().UnixNano()) + "-" + fmtInt64(int64(os.Getpid()))
}

// marshalOrNil возвращает JSON или nil
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
