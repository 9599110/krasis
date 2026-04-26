package systemconfig

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemConfig struct {
	ID          uuid.UUID       `json:"id"`
	ConfigKey   string          `json:"config_key"`
	ConfigValue json.RawMessage `json:"config_value"`
	Description string          `json:"description"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetAll(ctx context.Context) ([]SystemConfig, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, config_key, config_value, description FROM system_config ORDER BY config_key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []SystemConfig
	for rows.Next() {
		var c SystemConfig
		if err := rows.Scan(&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	if configs == nil {
		configs = []SystemConfig{}
	}
	return configs, nil
}

func (r *Repository) GetByKey(ctx context.Context, key string) (*SystemConfig, error) {
	var c SystemConfig
	err := r.pool.QueryRow(ctx,
		"SELECT id, config_key, config_value, description FROM system_config WHERE config_key = $1", key).
		Scan(&c.ID, &c.ConfigKey, &c.ConfigValue, &c.Description)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UpdateValue(ctx context.Context, key string, value interface{}) error {
	valJSON, _ := json.Marshal(map[string]interface{}{"value": value})
	tag, err := r.pool.Exec(ctx,
		"UPDATE system_config SET config_value = $1, updated_at = NOW() WHERE config_key = $2",
		valJSON, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Insert if not exists
		_, err = r.pool.Exec(ctx,
			"INSERT INTO system_config (config_key, config_value) VALUES ($1, $2)",
			key, valJSON)
	}
	return err
}

func (r *Repository) UpdateBatch(ctx context.Context, values map[string]interface{}) error {
	for key, value := range values {
		if err := r.UpdateValue(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// ConfigData represents the flattened config values for API responses/requests
type ConfigData struct {
	SiteName               string `json:"site_name"`
	AllowSignup            bool   `json:"allow_signup"`
	RequireEmailVerification bool   `json:"require_email_verification"`
	DefaultRole            string `json:"default_role"`
	MaxNotesPerUser        int    `json:"max_notes_per_user"`
	MaxStoragePerUserBytes int64  `json:"max_storage_per_user_bytes"`
	MaxFileSizeBytes       int64  `json:"max_file_size_bytes"`
	SessionDurationDays    int    `json:"session_duration_days"`
	MaxDevicesPerUser      int    `json:"max_devices_per_user"`
	EnableSharing          bool   `json:"enable_sharing"`
	EnableAI               bool   `json:"enable_ai"`
	MaintenanceMode        bool   `json:"maintenance_mode"`
}

func (r *Repository) GetAsConfigData(ctx context.Context) (*ConfigData, error) {
	configs, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	data := &ConfigData{}
	for _, c := range configs {
		var val map[string]interface{}
		json.Unmarshal(c.ConfigValue, &val)
		v, ok := val["value"]
		if !ok {
			continue
		}
		switch c.ConfigKey {
		case "site_name":
			if s, ok := v.(string); ok {
				data.SiteName = s
			}
		case "allow_signup":
			if b, ok := v.(bool); ok {
				data.AllowSignup = b
			}
		case "require_email_verification":
			if b, ok := v.(bool); ok {
				data.RequireEmailVerification = b
			}
		case "default_role":
			if s, ok := v.(string); ok {
				data.DefaultRole = s
			}
		case "max_notes_per_user":
			if f, ok := v.(float64); ok {
				data.MaxNotesPerUser = int(f)
			}
		case "max_storage_per_user_bytes":
			if f, ok := v.(float64); ok {
				data.MaxStoragePerUserBytes = int64(f)
			}
		case "max_file_size_bytes":
			if f, ok := v.(float64); ok {
				data.MaxFileSizeBytes = int64(f)
			}
		case "session_duration_days":
			if f, ok := v.(float64); ok {
				data.SessionDurationDays = int(f)
			}
		case "max_devices_per_user":
			if f, ok := v.(float64); ok {
				data.MaxDevicesPerUser = int(f)
			}
		case "enable_sharing":
			if b, ok := v.(bool); ok {
				data.EnableSharing = b
			}
		case "enable_ai":
			if b, ok := v.(bool); ok {
				data.EnableAI = b
			}
		case "maintenance_mode":
			if b, ok := v.(bool); ok {
				data.MaintenanceMode = b
			}
		}
	}
	return data, nil
}

// Ensure sql import is used (for sql.NullString)
var _ = sql.NullString{}
