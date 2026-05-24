package contracts

import (
	"testing"
)

func TestRecallContract_ReadRecall_NotFound(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &RecallContract{}

	_, err := contract.ReadRecall(ctx, "nonexistent-recall")
	if err == nil {
		t.Fatal("expected error when reading nonexistent recall, got nil")
	}
}
