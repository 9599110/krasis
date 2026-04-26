package admin

import (
	"testing"
)

func TestAdminHandler_NewHandler(t *testing.T) {
	// Verify handler can be created (compile-time check)
	// Real integration tests would need a full DB setup
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if h == nil {
		t.Fatal("handler should not be nil")
	}
}

func TestStatsOverview_Struct(t *testing.T) {
	s := StatsOverview{
		TotalUsers:    100,
		ActiveUsers:   50,
		TotalNotes:    500,
		TotalShares:   10,
		PendingShares: 3,
		StorageUsed:   1.5,
	}
	if s.TotalUsers != 100 {
		t.Fatalf("total users want 100 got %d", s.TotalUsers)
	}
	if s.StorageUsed != 1.5 {
		t.Fatalf("storage want 1.5 got %f", s.StorageUsed)
	}
}
