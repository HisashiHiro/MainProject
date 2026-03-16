package grpc

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor проверяет JWT-токен в метаданных и добавляет userID в контекст
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Пропускаем метод Login, если он есть (в вашем proto нет, но на будущее)
	if strings.Contains(info.FullMethod, "Login") {
		return handler(ctx, req)
	}

	// Получаем метаданные из контекста
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Извлекаем заголовок authorization
	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	// Токен должен быть в формате "Bearer <token>"
	tokenString := authHeader[0]
	if len(tokenString) > 7 && strings.ToLower(tokenString[:7]) == "bearer " {
		tokenString = tokenString[7:]
	} else {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
	}

	// Парсим и проверяем токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
	}

	// Извлекаем userID из claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
	}
	userID, ok := claims["user_id"].(float64) // JSON числа приходят как float64
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "user_id not found in token")
	}

	// Добавляем userID в контекст для использования в хендлерах
	newCtx := context.WithValue(ctx, "user_id", int64(userID))

	// Вызываем следующий обработчик
	return handler(newCtx, req)
}
