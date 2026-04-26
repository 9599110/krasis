package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenRevoked      = errors.New("token has been revoked")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)

type Claims struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	JTI       string `json:"jti"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
	redis         *redis.Client
}

func NewJWTManager(secret string, tokenDuration time.Duration, issuer string, redis *redis.Client) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secret),
		tokenDuration: tokenDuration,
		issuer:        issuer,
		redis:         redis,
	}
}

func (m *JWTManager) Generate(userID, role, sessionID string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		Role:      role,
		SessionID: sessionID,
		JTI:       uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check blacklist
	if m.IsBlacklisted(claims.JTI) {
		return nil, ErrTokenRevoked
	}

	return claims, nil
}

func (m *JWTManager) Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt_blacklist:%s", jti)
	return m.redis.Set(ctx, key, "1", ttl).Err()
}

func (m *JWTManager) IsBlacklisted(jti string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	key := fmt.Sprintf("jwt_blacklist:%s", jti)
	val, err := m.redis.Get(ctx, key).Result()
	return err == nil && val == "1"
}
