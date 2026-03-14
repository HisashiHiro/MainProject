package service

import (
	"MainProject/internal/model"
	"context"
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// UserRepository определяет методы для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	UpdateLastLogin(ctx context.Context, userID int64) error
}

// Ошибки домена пользователей
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// UserService описывает бизнес-логику пользователей
type UserService interface {
	Register(ctx context.Context, username, email, password string) (*model.User, error)
	Login(ctx context.Context, login, password string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
}

type userServiceImpl struct {
	repo UserRepository
}

// NewUserService создаёт новый сервис пользователей
func NewUserService(repo UserRepository) UserService {
	return &userServiceImpl{repo: repo}
}

// Register регистрирует нового пользователя
func (s *userServiceImpl) Register(ctx context.Context, username, email, password string) (*model.User, error) {
	// Проверка на существование
	existing, err := s.repo.GetUserByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}
	existing, err = s.repo.GetUserByLogin(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// Хеширование пароля
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := model.NewUser(username, email, hash)
	user.IsGenerated = false // создан пользователем

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login проверяет учётные данные и возвращает пользователя
func (s *userServiceImpl) Login(ctx context.Context, login, password string) (*model.User, error) {
	log.Printf("[DEBUG] Login attempt with login: %s", login)

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] GetUserByLogin error: %v", err)
		return nil, err
	}
	if user == nil {
		log.Printf("[DEBUG] User not found for login: %s", login)
		return nil, ErrInvalidCredentials
	}

	log.Printf("[DEBUG] User found: ID=%d, username=%s, email=%s", user.ID, user.Username, user.Email)

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Printf("[DEBUG] Password comparison failed: %v", err)
		return nil, ErrInvalidCredentials
	}

	log.Printf("[DEBUG] Login successful for user %d", user.ID)
	_ = s.repo.UpdateLastLogin(ctx, user.ID)
	return user, nil
}

/*
func (s *userServiceImpl) Login(ctx context.Context, login, password string) (*model.User, error) {
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Сравнение хеша пароля
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Обновляем время последнего входа
	_ = s.repo.UpdateLastLogin(ctx, user.ID) // игнорируем ошибку обновления

	return user, nil
}
*/
// GetUserByID возвращает пользователя по ID
func (s *userServiceImpl) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
