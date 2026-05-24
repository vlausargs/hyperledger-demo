package validation_test

import (
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"PROD-001", true},
		{"abc_123", true},
		{"valid-id-with-tilde~001", true},
		{"", false},
		{"a", true},
		{string(make([]byte, 65)), false},
		{"invalid id with spaces", false},
		{"inject'; DROP TABLE--", false},
	}
	for _, tt := range tests {
		err := validation.ValidateID(tt.id)
		if tt.valid && err != nil {
			t.Errorf("ValidateID(%q) should be valid, got error: %v", tt.id, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateID(%q) should be invalid, got nil", tt.id)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	err := validation.ValidateRequired("name", "")
	if err == nil {
		t.Error("expected error for empty required field")
	}

	err = validation.ValidateRequired("name", "value")
	if err != nil {
		t.Errorf("expected nil for non-empty field, got: %v", err)
	}
}

func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		from, to string
		valid    bool
	}{
		{"ACTIVE", "SHIPPED", true},
		{"SHIPPED", "DELIVERED", true},
		{"DELIVERED", "SOLD", true},
		{"ACTIVE", "RECALLED", true},
		{"SHIPPED", "ACTIVE", false},
		{"SOLD", "ACTIVE", false},
		{"DELIVERED", "SHIPPED", false},
	}
	for _, tt := range tests {
		err := validation.ValidateProductStatusTransition(tt.from, tt.to)
		if tt.valid && err != nil {
			t.Errorf("%s->%s should be valid, got: %v", tt.from, tt.to, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("%s->%s should be invalid", tt.from, tt.to)
		}
	}
}
