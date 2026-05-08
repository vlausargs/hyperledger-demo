package rest

import (
	"fmt"
	"regexp"
)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_\-~]{1,128}$`)

func validateID(kind, id string) error {
	if id == "" {
		return fmt.Errorf("%s ID required", kind)
	}
	if !validID.MatchString(id) {
		return fmt.Errorf("invalid %s ID: alphanumeric, underscore, hyphen, tilde only, max 128 chars", kind)
	}
	return nil
}

func validateProductID(id string) error  { return validateID("product", id) }
func validateShipmentID(id string) error { return validateID("shipment", id) }
func validateEventID(id string) error    { return validateID("event", id) }
func validateRecallID(id string) error   { return validateID("recall", id) }
