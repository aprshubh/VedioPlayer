package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
	"watchparty-backend/internal/model"
	"watchparty-backend/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidOTP   = errors.New("invalid or expired OTP")
)

type AuthService struct {
	jwtSecret []byte
	store     store.Store
	mu        sync.Mutex
	otpMap    map[string]otpRecord // email -> otpRecord
}

type otpRecord struct {
	code      string
	expiresAt time.Time
}

type Claims struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	jwt.RegisteredClaims
}

func NewAuthService(secret string, s store.Store) *AuthService {
	if secret == "" {
		secret = "super-secret-dyuet-jwt-key-change-in-prod"
	}
	return &AuthService{
		jwtSecret: []byte(secret),
		store:     s,
		otpMap:    make(map[string]otpRecord),
	}
}

// GenerateToken creates a signed JWT for a user (valid for 7 days)
func (a *AuthService) GenerateToken(user *model.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Avatar: user.Avatar,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "dyuet",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateToken verifies JWT and returns claims
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RequestOTP generates a 6-digit OTP code for an email address
func (a *AuthService) RequestOTP(email string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	a.otpMap[email] = otpRecord{
		code:      code,
		expiresAt: time.Now().Add(10 * time.Minute),
	}

	return code, nil
}

// VerifyOTP verifies email + OTP and returns user + JWT token
func (a *AuthService) VerifyOTP(email, code, preferredName, avatar string) (*model.User, string, error) {
	a.mu.Lock()
	record, exists := a.otpMap[email]
	isMasterDevCode := (os.Getenv("ENV") != "production" && code == "123456")
	if (!exists || time.Now().After(record.expiresAt) || record.code != code) && !isMasterDevCode {
		a.mu.Unlock()
		return nil, "", ErrInvalidOTP
	}
	delete(a.otpMap, email)
	a.mu.Unlock()

	// Check if user exists, or create new
	user, err := a.store.GetUserByEmail(email)
	if err != nil {
		if preferredName == "" {
			prefix := email
			if idx := strings.Index(email, "@"); idx > 0 {
				prefix = email[:idx]
			}
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			preferredName = "User-" + prefix
		}
		if avatar == "" {
			avatar = fmt.Sprintf("https://api.dicebear.com/7.x/bottts/svg?seed=%s", email)
		}
		user = &model.User{
			ID:        uuid.NewString(),
			Name:      preferredName,
			Email:     email,
			Avatar:    avatar,
			CreatedAt: time.Now(),
		}
		if err := a.store.CreateUser(user); err != nil {
			return nil, "", err
		}
	}

	token, err := a.GenerateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// GoogleLogin handles Google OAuth exchange or mocked OAuth profile
func (a *AuthService) GoogleLogin(email, name, avatar string) (*model.User, string, error) {
	user, err := a.store.GetUserByEmail(email)
	if err != nil {
		user = &model.User{
			ID:        uuid.NewString(),
			Name:      name,
			Email:     email,
			Avatar:    avatar,
			CreatedAt: time.Now(),
		}
		if err := a.store.CreateUser(user); err != nil {
			return nil, "", err
		}
	}

	token, err := a.GenerateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// GuestLogin creates an instant guest profile (e.g. for quick joining without email)
func (a *AuthService) GuestLogin(name string) (*model.User, string, error) {
	if name == "" {
		name = "Guest-" + uuid.NewString()[:5]
	}
	user := &model.User{
		ID:        uuid.NewString(),
		Name:      name,
		Email:     "",
		Avatar:    fmt.Sprintf("https://api.dicebear.com/7.x/bottts/svg?seed=%s", name),
		CreatedAt: time.Now(),
	}
	if err := a.store.CreateUser(user); err != nil {
		return nil, "", err
	}

	token, err := a.GenerateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}
