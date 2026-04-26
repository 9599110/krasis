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

func (n NullUUID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.UUID, nil
}
