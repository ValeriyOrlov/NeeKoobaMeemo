package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/auth/internal/model"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type UserRepository interface {
	Create(ctx context.Context, email, username, passwordHash string) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id uint) (model.User, error)
	CreateUnverifiedUser(ctx context.Context, user *model.User) error
	VerifyUser(ctx context.Context, token string) error
	FindByVerificationToken(ctx context.Context, token string) (model.User, error)
}

type GormUserRepo struct {
	db *gorm.DB
}

func NewGormUserRepo(db *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: db}
}

func (r *GormUserRepo) Create(ctx context.Context, email, username, passwordHash string) (model.User, error) {
	user := model.User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}
	result := r.db.WithContext(ctx).Create(&user)
	// тут ловим дубликат или остальные ошибки БД
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return model.User{}, ErrUserAlreadyExists
		}
		return model.User{}, fmt.Errorf("create user in repository: %w", result.Error)
	}

	return user, nil
}

func (r *GormUserRepo) FindByEmail(ctx context.Context, email string) (model.User, error) {
	user := model.User{}
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user by email: %w", result.Error)
	}
	return user, nil
}

func (r *GormUserRepo) FindByID(ctx context.Context, id uint) (model.User, error) {
	user := model.User{}
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user by id: %w", result.Error)
	}
	return user, nil
}

// Создаём неподтверждённого пользователя
func (r *GormUserRepo) CreateUnverifiedUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Активируем пользователя по токену
func (r *GormUserRepo) VerifyUser(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("verification_token = ?", token).
		Updates(map[string]interface{}{
			"is_verified":        true,
			"verification_token": nil, // очищаем токен после использования
		})
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return result.Error
}

// Находим пользователя по verification_token
func (r *GormUserRepo) FindByVerificationToken(ctx context.Context, token string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("verification_token = ?", token).First(&user).Error
	return user, err
}
