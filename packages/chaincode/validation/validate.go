package validation

import (
	"fmt"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-~]{1,64}$`)

func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid ID %q: must be 1-64 alphanumeric/underscore/hyphen/tilde characters", id)
	}
	return nil
}

func ValidateRequired(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

var productTransitions = map[string]map[string]bool{
	"ACTIVE":    {"SHIPPED": true, "RECALLED": true, "SCRAPPED": true},
	"SHIPPED":   {"DELIVERED": true, "RECALLED": true},
	"DELIVERED": {"SOLD": true, "RECALLED": true, "ACTIVE": true},
	"RECALLED":  {"SCRAPPED": true},
}

func ValidateProductStatusTransition(from, to string) error {
	allowed, exists := productTransitions[from]
	if !exists {
		return fmt.Errorf("unknown product status %q", from)
	}
	if !allowed[to] {
		return fmt.Errorf("invalid status transition: %s -> %s", from, to)
	}
	return nil
}

var shipmentTransitions = map[string]map[string]bool{
	"DRAFT":      {"IN_TRANSIT": true, "CANCELLED": true},
	"IN_TRANSIT": {"DELIVERED": true, "RECALLED": true},
}

func ValidateShipmentStatusTransition(from, to string) error {
	allowed, exists := shipmentTransitions[from]
	if !exists {
		return fmt.Errorf("unknown shipment status %q", from)
	}
	if !allowed[to] {
		return fmt.Errorf("invalid shipment status transition: %s -> %s", from, to)
	}
	return nil
}

func ValidateSelectorValue(value string) error {
	for _, ch := range value {
		if ch == '"' || ch == '\\' || ch == '{' || ch == '}' {
			return fmt.Errorf("invalid character %q in query parameter", ch)
		}
	}
	return nil
}
