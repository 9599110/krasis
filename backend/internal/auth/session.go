package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	SessionID  string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	DeviceName string    `json:"device_name"`
	DeviceType string    `json:"device_type"`
	LastActive time.Time `json:"last_active_at"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsCurrent  bool      `json:"is_current"`
}

type SessionInfo struct {
	UserAgent string
	IPAddress string
}

type SessionManager struct {
	redis    *redis.Client
	duration time.Duration
}

func NewSessionManager(redis *redis.Client, duration time.Duration) *SessionManager {
	return &SessionManager{
		redis:    redis,
		duration: duration,
	}
}

func (m *SessionManager) Create(ctx context.Context, userID string, info *SessionInfo) (*Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	session := &Session{
		SessionID:  sessionID,
		UserID:     userID,
		UserAgent:  info.UserAgent,
		IPAddress:  info.IPAddress,
		DeviceName: detectDeviceName(info.UserAgent),
		DeviceType: detectDeviceType(info.UserAgent),
		LastActive: now,
		CreatedAt:  now,
		ExpiresAt:  now.Add(m.duration),
	}

	key := fmt.Sprintf("session:%s", sessionID)
	pipe := m.redis.Pipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"user_id":      session.UserID,
		"user_agent":   session.UserAgent,
		"ip_address":   session.IPAddress,
		"device_name":  session.DeviceName,
		"device_type":  session.DeviceType,
		"last_active":  session.LastActive.Format(time.RFC3339),
		"created_at":   session.CreatedAt.Format(time.RFC3339),
		"expires_at":   session.ExpiresAt.Format(time.RFC3339),
	})
	pipe.Expire(ctx, key, m.duration)
	pipe.SAdd(ctx, fmt.Sprintf("user_sessions:%s", userID), sessionID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

func (m *SessionManager) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := m.redis.HGetAll(ctx, key).Result()
	if err != nil || len(data) == 0 {
		return nil, ErrSessionNotFound
	}

	return parseSession(data), nil
}

func (m *SessionManager) Delete(ctx context.Context, sessionID string) error {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	pipe := m.redis.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("session:%s", sessionID))
	pipe.SRem(ctx, fmt.Sprintf("user_sessions:%s", session.UserID), sessionID)
	_, err = pipe.Exec(ctx)
	return err
}

func (m *SessionManager) GetUserSessions(ctx context.Context, userID string, currentSessionID string) ([]*Session, error) {
	sessionIDs, err := m.redis.SMembers(ctx, fmt.Sprintf("user_sessions:%s", userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get user sessions: %w", err)
	}

	sessions := make([]*Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, err := m.Get(ctx, id)
		if err != nil {
			continue
		}
		if id == currentSessionID {
			session.IsCurrent = true
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (m *SessionManager) Refresh(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return m.redis.HSet(ctx, key, "last_active", time.Now().Format(time.RFC3339)).Err()
}

// GetUserSessionsMap returns sessions as maps (for interface compatibility).
func (m *SessionManager) GetUserSessionsMap(ctx context.Context, userID, currentSessionID string) ([]map[string]interface{}, error) {
	sessionIDs, err := m.redis.SMembers(ctx, fmt.Sprintf("user_sessions:%s", userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get user sessions: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, err := m.Get(ctx, id)
		if err != nil {
			continue
		}
		m := map[string]interface{}{
			"session_id":   session.SessionID,
			"user_id":      session.UserID,
			"device_name":  session.DeviceName,
			"device_type":  session.DeviceType,
			"ip_address":   session.IPAddress,
			"user_agent":   session.UserAgent,
			"last_active":  session.LastActive,
			"created_at":   session.CreatedAt,
			"expires_at":   session.ExpiresAt,
			"is_current":   id == currentSessionID,
		}
		result = append(result, m)
	}

	return result, nil
}

// DeleteSession is an adapter for the user.SessionManager interface.
func (m *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	return m.Delete(ctx, sessionID)
}

func (m *SessionManager) DeleteAllForUser(ctx context.Context, userID string) error {
	sessions, err := m.GetUserSessions(ctx, userID, "")
	if err != nil {
		return err
	}

	pipe := m.redis.Pipeline()
	for _, s := range sessions {
		pipe.Del(ctx, fmt.Sprintf("session:%s", s.SessionID))
	}
	pipe.Del(ctx, fmt.Sprintf("user_sessions:%s", userID))
	_, err = pipe.Exec(ctx)
	return err
}

func parseSession(data map[string]string) *Session {
	lastActive, _ := time.Parse(time.RFC3339, data["last_active"])
	createdAt, _ := time.Parse(time.RFC3339, data["created_at"])
	expiresAt, _ := time.Parse(time.RFC3339, data["expires_at"])

	return &Session{
		SessionID:  "",
		UserID:     data["user_id"],
		UserAgent:  data["user_agent"],
		IPAddress:  data["ip_address"],
		DeviceName: data["device_name"],
		DeviceType: data["device_type"],
		LastActive: lastActive,
		CreatedAt:  createdAt,
		ExpiresAt:  expiresAt,
	}
}

func detectDeviceName(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		if strings.Contains(ua, "iphone") {
			return "iPhone"
		}
		return "Android"
	}
	if strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os") {
		return "Mac"
	}
	if strings.Contains(ua, "windows") {
		return "Windows"
	}
	if strings.Contains(ua, "linux") {
		return "Linux"
	}
	return "Unknown Device"
}

func detectDeviceType(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "android") || strings.Contains(ua, "mobile") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}
