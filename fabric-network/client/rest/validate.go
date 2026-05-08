package rest

import (
	"fmt"
	"regexp"
)

var validAssetID = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,64}$`)

func validateAssetID(id string) error {
	if id == "" {
		return fmt.Errorf("asset ID required")
	}
	if !validAssetID.MatchString(id) {
		return fmt.Errorf("invalid asset ID: alphanumeric, underscore, hyphen only, max 64 chars")
	}
	return nil
}
