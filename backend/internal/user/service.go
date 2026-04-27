package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetOrCreateByOAuth(ctx context.Context, email, username, avatarURL string) (*User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// User not found, create new
		user, err = s.repo.Create(ctx, email, username, avatarURL)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		if err := s.repo.AssignDefaultRole(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("assign default role: %w", err)
		}
		return user, nil
	}
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) UpdateProfile(ctx context.Context, id uuid.UUID, username, avatarURL string) error {
	return s.repo.UpdateProfile(ctx, id, username, avatarURL)
}

func (s *UserService) GetRole(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.repo.GetRole(ctx, userID)
}

var validRoles = map[string]bool{"admin": true, "member": true, "viewer": true}

func (s *UserService) ListUsers(ctx context.Context, keyword, role string, page, size int) ([]*UserWithRole, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListUsers(ctx, keyword, role, page, size)
}

func (s *UserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	if !validRoles[roleName] {
		return fmt.Errorf("invalid role: %s", roleName)
	}
	return s.repo.UpdateRole(ctx, userID, roleName)
}

func (s *UserService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status int16) error {
	return s.repo.UpdateStatus(ctx, userID, status)
}

func (s *UserService) CreateUser(ctx context.Context, email, username, password, role string) (*User, error) {
	user, err := s.repo.Create(ctx, email, username, "")
	if err != nil {
		return nil, err
	}
	if role != "" && validRoles[role] {
		s.repo.UpdateRole(ctx, user.ID, role)
	} else {
		s.repo.AssignDefaultRole(ctx, user.ID)
	}
	return user, nil
}

func (s *UserService) RegisterLocal(ctx context.Context, email, username, password string) (*User, error) {
	if email == "" || username == "" || password == "" {
		return nil, fmt.Errorf("missing fields")
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("password too short")
	}

	// ensure email uniqueness
	if _, err := s.repo.GetByEmail(ctx, email); err == nil {
		return nil, fmt.Errorf("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateWithPasswordHash(ctx, email, username, string(hash), "")
	if err != nil {
		return nil, err
	}
	if err := s.repo.AssignDefaultRole(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("assign default role: %w", err)
	}
	return user, nil
}

func (s *UserService) AuthenticateByUsername(ctx context.Context, username, password string) (*User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if user.Status == 0 {
		return nil, fmt.Errorf("user disabled")
	}
	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return s.repo.Delete(ctx, userID)
}

func (s *UserService) BatchUpdateStatus(ctx context.Context, userIDs []uuid.UUID, status int16) error {
	return s.repo.BatchUpdateStatus(ctx, userIDs, status)
}

func (s *UserService) GetGroupID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return s.repo.GetGroupID(ctx, userID)
}

func (s *UserService) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
