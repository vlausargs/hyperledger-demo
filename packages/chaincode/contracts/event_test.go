package contracts

import (
	"testing"
)

func TestEventContract_GetEvents_Empty(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &EventContract{}

	// No events inserted; should return empty slice.
	events, err := contract.GetEvents(ctx, "prod-001")
	if err != nil {
		t.Fatalf("GetEvents returned unexpected error: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
