package auditlog

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuditLog_Model(t *testing.T) {
	adminID := uuid.New()
	targetID := uuid.NullUUID{UUID: uuid.New(), Valid: true}
	now := time.Now()
	log := AuditLog{
		ID:         uuid.New(),
		Action:     "user.update_role",
		TargetType: sql.NullString{String: "user", Valid: true},
		TargetID:   targetID,
		AdminID:    adminID,
		Changes:    json.RawMessage(`{"role": "admin"}`),
		IPAddress:  sql.NullString{String: "127.0.0.1", Valid: true},
		UserAgent:  sql.NullString{String: "Mozilla/5.0", Valid: true},
		CreatedAt:  now,
	}
	if log.Action != "user.update_role" {
		t.Fatalf("action want user.update_role got %s", log.Action)
	}
	if log.IPAddress.String != "127.0.0.1" {
		t.Fatalf("ip want 127.0.0.1 got %s", log.IPAddress.String)
	}
}

func TestAuditLog_JSONSerialization(t *testing.T) {
	log := AuditLog{
		ID:     uuid.New(),
		Action: "test.action",
		Changes: json.RawMessage(`{"key": "value"}`),
	}
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AuditLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Action != "test.action" {
		t.Fatalf("action want test.action got %s", decoded.Action)
	}
}

func TestAuditLog_NullFields(t *testing.T) {
	log := AuditLog{
		ID:         uuid.New(),
		Action:     "test.null",
		TargetType: sql.NullString{},
		TargetID:   uuid.NullUUID{},
	}
	if log.TargetType.Valid {
		t.Fatal("TargetType should not be valid")
	}
	if log.TargetID.Valid {
		t.Fatal("TargetID should not be valid")
	}
}
