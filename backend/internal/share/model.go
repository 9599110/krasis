package share

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrShareNotFound   = errors.New("share not found")
	ErrShareExpired    = errors.New("share has expired")
	ErrSharePending    = errors.New("share pending review")
	ErrShareRejected   = errors.New("share rejected")
	ErrInvalidPassword = errors.New("invalid password")
	ErrShareExists     = errors.New("share already exists")
)

type NoteShare struct {
	ID              uuid.UUID      `json:"id"`
	NoteID          uuid.UUID      `json:"note_id"`
	ShareToken      string         `json:"share_token"`
	ShareType       string         `json:"share_type"`
	Permission      string         `json:"permission"`
	PasswordHash    sql.NullString `json:"-"`
	ExpiresAt       sql.NullTime   `json:"expires_at"`
	Status          string         `json:"status"`
	ContentSnapshot sql.NullString `json:"-"`
	RejectionReason sql.NullString `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	CreatedBy       uuid.UUID      `json:"created_by"`
}

type ShareRepository struct {
	pool *pgxpool.Pool
}

func NewShareRepository(pool *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{pool: pool}
}

func (r *ShareRepository) Create(ctx context.Context, share *NoteShare) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO note_shares (note_id, share_token, share_type, permission, password_hash,
								  expires_at, status, content_snapshot, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, share.NoteID, share.ShareToken, share.ShareType, share.Permission,
		share.PasswordHash, share.ExpiresAt, share.Status, share.ContentSnapshot, share.CreatedBy,
	).Scan(&share.ID, &share.CreatedAt)
	return err
}

func (r *ShareRepository) GetByToken(ctx context.Context, token string) (*NoteShare, error) {
	var s NoteShare
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, share_token, share_type, permission, password_hash,
			   expires_at, status, content_snapshot, rejection_reason, created_at, created_by
		FROM note_shares WHERE share_token = $1
	`, token).Scan(
		&s.ID, &s.NoteID, &s.ShareToken, &s.ShareType, &s.Permission, &s.PasswordHash,
		&s.ExpiresAt, &s.Status, &s.ContentSnapshot, &s.RejectionReason, &s.CreatedAt, &s.CreatedBy,
	)
	if err != nil {
		return nil, ErrShareNotFound
	}
	return &s, nil
}

func (r *ShareRepository) GetByNoteID(ctx context.Context, noteID uuid.UUID) (*NoteShare, error) {
	var s NoteShare
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, share_token, share_type, permission, password_hash,
			   expires_at, status, content_snapshot, rejection_reason, created_at, created_by
		FROM note_shares WHERE note_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, noteID).Scan(
		&s.ID, &s.NoteID, &s.ShareToken, &s.ShareType, &s.Permission, &s.PasswordHash,
		&s.ExpiresAt, &s.Status, &s.ContentSnapshot, &s.RejectionReason, &s.CreatedAt, &s.CreatedBy,
	)
	if err != nil {
		return nil, ErrShareNotFound
	}
	return &s, nil
}

func (r *ShareRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, reason string) error {
	query := "UPDATE note_shares SET status = $1, reviewed_at = NOW(), reviewed_by = $2"
	args := []interface{}{status, reviewerID}
	idx := 3

	if reason != "" {
		query += fmt.Sprintf(", rejection_reason = $%d", idx)
		args = append(args, reason)
		idx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrShareNotFound
	}
	return nil
}

func (r *ShareRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM note_shares WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrShareNotFound
	}
	return nil
}

func generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}

func verifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// AdminShareListItem is a share list item enriched with user/note info for admin review
type AdminShareListItem struct {
	ID              uuid.UUID  `json:"id"`
	ShareToken      string     `json:"share_token"`
	NoteID          uuid.UUID  `json:"note_id"`
	NoteTitle       string     `json:"note_title"`
	NotePreview     string     `json:"note_preview"`
	ContentSnapshot string     `json:"content_snapshot,omitempty"`
	OwnerID         uuid.UUID  `json:"owner_id"`
	OwnerUsername   string     `json:"owner_username"`
	OwnerEmail      string     `json:"owner_email"`
	ShareType       string     `json:"share_type"`
	Permission      string     `json:"permission"`
	PasswordProtected bool     `json:"password_protected"`
	ExpiresAt       *time.Time `json:"expires_at"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}

// ListShares returns shares with user/note info for admin review.
// statusFilter can be "pending", "approved", "rejected", "revoked", or empty for all.
func (r *ShareRepository) ListShares(ctx context.Context, statusFilter, keyword string, page, size int) ([]AdminShareListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size

	// Build query
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if statusFilter != "" {
		where += fmt.Sprintf(" AND s.status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}

	if keyword != "" {
		where += fmt.Sprintf(" AND (n.title ILIKE $%d OR u.username ILIKE $%d OR u.email ILIKE $%d)", idx, idx, idx)
		args = append(args, "%"+keyword+"%")
		idx++
	}

	// Count query
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM note_shares s JOIN notes n ON n.id = s.note_id JOIN users u ON u.id = s.created_by %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with content_snapshot
	query := fmt.Sprintf(`
		SELECT s.id, s.note_id, s.share_token, s.share_type, s.permission,
			   s.password_hash IS NOT NULL AS password_protected, s.expires_at, s.status,
			   s.content_snapshot, s.created_at, s.reviewed_at,
			   s.rejection_reason,
			   n.title AS note_title,
			   u.id AS owner_id, u.username AS owner_username, u.email AS owner_email,
			   rev.username AS reviewed_by_username
		FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		JOIN users u ON u.id = s.created_by
		LEFT JOIN users rev ON rev.id = s.reviewed_by
		%s
		ORDER BY s.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []AdminShareListItem
	for rows.Next() {
		var item AdminShareListItem
		var expiresAt sql.NullTime
		var passwordProtected bool
		var contentSnapshot sql.NullString
		var reviewedAt sql.NullTime
		var reviewedBy sql.NullString
		var rejectionReason sql.NullString
		if err := rows.Scan(
			&item.ID, &item.NoteID, &item.ShareToken, &item.ShareType, &item.Permission,
			&passwordProtected, &expiresAt, &item.Status,
			&contentSnapshot, &item.CreatedAt, &reviewedAt,
			&rejectionReason,
			&item.NoteTitle,
			&item.OwnerID, &item.OwnerUsername, &item.OwnerEmail,
			&reviewedBy,
		); err != nil {
			return nil, 0, err
		}
		item.PasswordProtected = passwordProtected
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		if contentSnapshot.Valid {
			item.ContentSnapshot = contentSnapshot.String
			// Generate a short preview from the snapshot
			if len(item.ContentSnapshot) > 100 {
				item.NotePreview = item.ContentSnapshot[:100] + "..."
			} else {
				item.NotePreview = item.ContentSnapshot
			}
		}
		if reviewedAt.Valid {
			item.ReviewedAt = &reviewedAt.Time
		}
		if reviewedBy.Valid {
			item.ReviewedBy = reviewedBy.String
		}
		if rejectionReason.Valid {
			item.RejectionReason = rejectionReason.String
		}
		items = append(items, item)
	}
	if items == nil {
		items = []AdminShareListItem{}
	}
	return items, total, rows.Err()
}

// GetShareDetail returns a single share with full detail for admin review.
func (r *ShareRepository) GetShareDetail(ctx context.Context, shareID uuid.UUID) (*AdminShareListItem, error) {
	var item AdminShareListItem
	var expiresAt sql.NullTime
	var passwordProtected bool
	var contentSnapshot sql.NullString
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullString
	var rejectionReason sql.NullString

	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.note_id, s.share_token, s.share_type, s.permission,
			   s.password_hash IS NOT NULL AS password_protected, s.expires_at, s.status,
			   s.content_snapshot, s.created_at, s.reviewed_at,
			   s.rejection_reason,
			   n.title AS note_title,
			   u.id AS owner_id, u.username AS owner_username, u.email AS owner_email,
			   rev.username AS reviewed_by_username
		FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		JOIN users u ON u.id = s.created_by
		LEFT JOIN users rev ON rev.id = s.reviewed_by
		WHERE s.id = $1
	`, shareID).Scan(
		&item.ID, &item.NoteID, &item.ShareToken, &item.ShareType, &item.Permission,
		&passwordProtected, &expiresAt, &item.Status,
		&contentSnapshot, &item.CreatedAt, &reviewedAt,
		&rejectionReason,
		&item.NoteTitle,
		&item.OwnerID, &item.OwnerUsername, &item.OwnerEmail,
		&reviewedBy,
	)
	if err != nil {
		return nil, ErrShareNotFound
	}

	item.PasswordProtected = passwordProtected
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if contentSnapshot.Valid {
		item.ContentSnapshot = contentSnapshot.String
		if len(item.ContentSnapshot) > 100 {
			item.NotePreview = item.ContentSnapshot[:100] + "..."
		} else {
			item.NotePreview = item.ContentSnapshot
		}
	}
	if reviewedAt.Valid {
		item.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		item.ReviewedBy = reviewedBy.String
	}
	if rejectionReason.Valid {
		item.RejectionReason = rejectionReason.String
	}
	return &item, nil
}

// GetShareStats returns counts by status.
type ShareStats struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Approved  int64 `json:"approved"`
	Rejected  int64 `json:"rejected"`
	Revoked   int64 `json:"revoked"`
}

func (r *ShareRepository) GetShareStats(ctx context.Context) (*ShareStats, error) {
	stats := &ShareStats{}
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*),
			   COUNT(*) FILTER (WHERE status = 'pending'),
			   COUNT(*) FILTER (WHERE status = 'approved'),
			   COUNT(*) FILTER (WHERE status = 'rejected'),
			   COUNT(*) FILTER (WHERE status = 'revoked')
		FROM note_shares
	`).Scan(&stats.Total, &stats.Pending, &stats.Approved, &stats.Rejected, &stats.Revoked)
	return stats, err
}

// BatchUpdateStatus updates the status of multiple shares.
func (r *ShareRepository) BatchUpdateStatus(ctx context.Context, shareIDs []uuid.UUID, status string, reviewerID uuid.UUID, reason string) error {
	query := "UPDATE note_shares SET status = $1, reviewed_at = NOW(), reviewed_by = $2"
	args := []interface{}{status, reviewerID}
	idx := 3

	if reason != "" {
		query += fmt.Sprintf(", rejection_reason = $%d", idx)
		args = append(args, reason)
		idx++
	}

	query += fmt.Sprintf(" WHERE id = ANY($%d)", idx)
	args = append(args, shareIDs)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}
