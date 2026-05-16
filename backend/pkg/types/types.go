package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NullTime is a time.Time that can be NULL in the database and marshals to JSON null when invalid.
type NullTime struct {
	Time  time.Time
	Valid bool
}

func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	if t, ok := value.(time.Time); ok {
		n.Time = t
		n.Valid = true
		return nil
	}
	return fmt.Errorf("cannot scan %T into NullTime", value)
}

func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Time)
}

type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

func (n *NullUUID) Scan(value interface{}) error {
	if value == nil {
		n.UUID, n.Valid = uuid.Nil, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case []byte:
		id, err := uuid.ParseBytes(v)
		if err != nil {
			return err
		}
		n.UUID = id
		return nil
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		n.UUID = id
		return nil
	case uuid.UUID:
		n.UUID = v
		return nil
	}
	return fmt.Errorf("cannot scan %T into NullUUID", value)
}

func (n NullUUID) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.UUID)
}

func (n NullUUID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.UUID, nil
}

// NullString is like sql.NullString but marshals to JSON as a plain string (or null).
type NullString struct {
	String string
	Valid  bool
}

func (n *NullString) Scan(value interface{}) error {
	if value == nil {
		n.String, n.Valid = "", false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case string:
		n.String = v
	case []byte:
		n.String = string(v)
	default:
		n.String = fmt.Sprint(value)
	}
	return nil
}

func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.String, nil
}

func (n NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.String, n.Valid = "", false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.String)
}

// NullInt64 is like sql.NullInt64 but marshals to JSON as a plain int64 (or null).
type NullInt64 struct {
	Int64 int64
	Valid bool
}

func (n *NullInt64) Scan(value interface{}) error {
	if value == nil {
		n.Int64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case int64:
		n.Int64 = v
	case float64:
		n.Int64 = int64(v)
	case int:
		n.Int64 = int64(v)
	case []byte:
		fmt.Sscanf(string(v), "%d", &n.Int64)
	default:
		n.Int64 = 0
	}
	return nil
}

func (n NullInt64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int64, nil
}

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Int64)
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Int64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Int64)
}

// NullInt32 is like sql.NullInt32 but marshals to JSON as a plain int32 (or null).
type NullInt32 struct {
	Int32 int32
	Valid bool
}

func (n *NullInt32) Scan(value interface{}) error {
	if value == nil {
		n.Int32, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case int32:
		n.Int32 = v
	case int64:
		n.Int32 = int32(v)
	case float64:
		n.Int32 = int32(v)
	case []byte:
		fmt.Sscanf(string(v), "%d", &n.Int32)
	default:
		n.Int32 = 0
	}
	return nil
}

func (n NullInt32) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int32, nil
}

func (n NullInt32) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Int32)
}

func (n *NullInt32) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Int32, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Int32)
}

// NullFloat64 is like sql.NullFloat64 but marshals to JSON as a plain float64 (or null).
type NullFloat64 struct {
	Float64 float64
	Valid   bool
}

func (n *NullFloat64) Scan(value interface{}) error {
	if value == nil {
		n.Float64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case float64:
		n.Float64 = v
	case float32:
		n.Float64 = float64(v)
	case int64:
		n.Float64 = float64(v)
	case []byte:
		fmt.Sscanf(string(v), "%f", &n.Float64)
	default:
		n.Float64 = 0
	}
	return nil
}

func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Float64, nil
}

func (n NullFloat64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Float64)
}

func (n *NullFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Float64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Float64)
}
