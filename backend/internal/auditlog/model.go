package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLog struct {
	ID            uuid.UUID       `json:"id"`
	Action        string          `json:"action"`
	TargetType    sql.NullString  `json:"target_type"`
	TargetID      uuid.NullUUID   `json:"target_id"`
	AdminID       uuid.UUID       `json:"admin_id"`
	AdminUsername sql.NullString  `json:"admin_username"`
	Changes       json.RawMessage `json:"changes,omitempty"`
	IPAddress     sql.NullString  `json:"ip_address"`
	UserAgent     sql.NullString  `json:"user_agent"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, action, userID string, startDate, endDate string, page, size int) ([]AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if action != "" {
		where += " AND action LIKE $1"
		args = append(args, "%"+action+"%")
		idx++
	}
	if userID != "" {
		where += " AND admin_id = $" + string(rune('0'+idx))
		args = append(args, userID)
		idx++
	}
	if startDate != "" {
		where += " AND created_at >= $" + string(rune('0'+idx))
		args = append(args, startDate)
		idx++
	}
	if endDate != "" {
		where += " AND created_at <= $" + string(rune('0'+idx))
		args = append(args, endDate)
		idx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM audit_logs " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT id, action, target_type, target_id, admin_id, admin_username, changes, ip_address, user_agent, created_at FROM audit_logs " + where + " ORDER BY created_at DESC LIMIT $" + string(rune('0'+idx)) + " OFFSET $" + string(rune('0'+idx+1))
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(&log.ID, &log.Action, &log.TargetType, &log.TargetID, &log.AdminID, &log.AdminUsername, &log.Changes, &log.IPAddress, &log.UserAgent, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}
	if logs == nil {
		logs = []AuditLog{}
	}
	return logs, total, rows.Err()
}

func (r *Repository) Create(ctx context.Context, log *AuditLog) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (action, target_type, target_id, admin_id, admin_username, changes, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.Action, log.TargetType, log.TargetID, log.AdminID, log.AdminUsername, log.Changes, log.IPAddress, log.UserAgent)
	return err
}
