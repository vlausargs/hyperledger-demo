package contracts

import (
	"testing"
)

func TestSaleContract_ReadSale_NotFound(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &SaleContract{}

	_, err := contract.ReadSale(ctx, "nonexistent-sale")
	if err == nil {
		t.Fatal("expected error when reading nonexistent sale, got nil")
	}
}
