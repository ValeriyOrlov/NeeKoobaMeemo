package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/model"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

var (
	ErrInvalidInput             = errors.New("invalid input")
	ErrInvalidEmail             = errors.New("invalid email")
	ErrWeakPassword             = errors.New("min 8 characters")
	ErrShortUsername            = errors.New("min 3 characters")
	ErrUserAlreadyExists        = errors.New("user already exists")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrUserNotFound             = errors.New("user not found")
	ErrInvalidVerificationToken = errors.New("invalid verification token")
	ErrEmailNotVerified         = errors.New("email not verified")
)

type AuthService struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.TokenRepository
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
	emailConfig EmailConfig
}

type EmailConfig struct {
	From     string
	SMTPHost string
	SMTPPort int
	Username string
	Password string
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	emailConfig EmailConfig,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		jwtSecret:   jwtSecret,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		emailConfig: emailConfig,
	}
}

func (s *AuthService) Register(ctx context.Context, email, username, password string) error {
	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}
	if len(username) < 3 {
		return ErrShortUsername
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	token := uuid.New().String()

	user := &model.User{
		Email:             email,
		Username:          username,
		PasswordHash:      string(hashedPassword),
		IsVerified:        false,
		VerificationToken: token,
	}

	// Сохраняем пользователя в БД
	if err := s.userRepo.CreateUnverifiedUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	// Отправляем письмо
	if err := s.sendVerificationEmail(email, token); err != nil {
		// Логируем ошибку, но не прерываем процесс
		log.Printf("send verification email: %v", err)
	}

	return nil
}

// sendVerificationEmail отправляет письмо с токеном подтверждения
func (s *AuthService) sendVerificationEmail(to, token string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.emailConfig.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Подтверждение регистрации в NeeKoobaMeemo!")

	link := fmt.Sprintf("http://localhost:8081/verify?token=%s", token)
	body := fmt.Sprintf(`
        <h2>Добро пожаловать в NeeKoobaMeemo!</h2>
        <p>Перейдите по ссылке, чтобы подтвердить email:</p>
        <a href="%s">%s</a>
        <p>Если вы не регистрировались, просто проигнорируйте это письмо.</p>
    `, link, link)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.emailConfig.SMTPHost, s.emailConfig.SMTPPort, s.emailConfig.Username, s.emailConfig.Password)
	return d.DialAndSend(m)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("find user by email: %w", err)
	}

	// Проверка подтверждения почты
	if !user.IsVerified {
		return "", "", ErrEmailNotVerified
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("comparing hash and password: %w", err)
	}

	accessToken, refreshToken, err := s.createTokenPair(ctx, user.ID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("create token pair error: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr string) (newAccess, newRefresh string, err error) {
	token, err := jwt.Parse(refreshTokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", ErrInvalidRefreshToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", ErrInvalidRefreshToken
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return "", "", ErrInvalidRefreshToken
	}
	userID := uint(userIDFloat)

	// Ищем в базе
	_, err = s.tokenRepo.FindByToken(ctx, refreshTokenStr)
	if errors.Is(err, repository.ErrTokenNotFound) {
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		return "", "", fmt.Errorf("refresh: find token: %w", err)
	}

	// Удаляем старый
	if err := s.tokenRepo.DeleteByToken(ctx, refreshTokenStr); err != nil {
		return "", "", fmt.Errorf("refresh: delete old token: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", ErrUserNotFound
		}
		return "", "", fmt.Errorf("find user by id error: %w", err)
	}

	accessToken, refreshToken, err := s.createTokenPair(ctx, userID, user.Username)
	if err != nil {
		return "", "", fmt.Errorf("create token pair error: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) createTokenPair(ctx context.Context, userID uint, username string) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(s.accessTTL).Unix(),
		"iat":      time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	jti := uuid.New().String()
	refreshExpAt := time.Now().Add(s.refreshTTL)
	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     refreshExpAt.Unix(),
		"iat":     time.Now().Unix(),
		"jti":     jti,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	refreshModel := model.RefreshToken{
		UserID:    userID,
		Token:     signedRefresh,
		ExpiresAt: refreshExpAt,
	}

	err = s.tokenRepo.Create(ctx, &refreshModel)
	if err != nil {
		return "", "", fmt.Errorf("adding refresh token to bd: %w", err)

	}
	return signedAccess, signedRefresh, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	_, err := s.tokenRepo.FindByToken(ctx, refreshTokenStr)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil
		}
		return fmt.Errorf("logout error: %w", err)
	}
	if err := s.tokenRepo.DeleteByToken(ctx, refreshTokenStr); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) (string, string, *model.User, error) {
	// Находим пользователя по токену
	user, err := s.userRepo.FindByVerificationToken(ctx, token)
	if err != nil {
		return "", "", nil, ErrInvalidVerificationToken
	}

	// Активируем пользователя
	if err := s.userRepo.VerifyUser(ctx, token); err != nil {
		return "", "", nil, fmt.Errorf("verify user: %w", err)
	}

	// Генерируем пару токенов (как в Login)
	accessToken, refreshToken, err := s.createTokenPair(ctx, user.ID, user.Username)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, &user, nil
}
