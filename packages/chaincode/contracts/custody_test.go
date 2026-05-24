package contracts

import (
	"testing"
)

func TestCustodyContract_GetCustodyChain_Empty(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &CustodyContract{}

	// No custody records inserted; should return empty slice.
	records, err := contract.GetCustodyChain(ctx, "ship-001")
	if err != nil {
		t.Fatalf("GetCustodyChain returned unexpected error: %v", err)
	}
	if records == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(records) != 0 {
		t.Errorf("expected 0 custody records, got %d", len(records))
	}
}
