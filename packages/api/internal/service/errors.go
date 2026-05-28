package service

import (
	"strings"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// mapFabricError converts a raw Fabric/chaincode error into a typed
// *errors.AppError. The Fabric Gateway SDK surfaces chaincode rejection
// messages as plain strings inside wrapped errors, so we sniff for known
// substrings. resource is used to build a user-facing message (e.g.
// "product not found").
//
// Mapping rules:
//   - "does not exist"     -> 404 NotFound
//   - "already exists"     -> 409 Conflict
//   - "not authorized" / "is not authorized" / "access denied" -> 403 Forbidden
//   - "unauthorized"       -> 401 Unauthorized
//   - validation phrases   -> 400 BadRequest
//     ("invalid", "must be", "required", "not DELIVERED",
//      "not owned by", "under recall")
//   - everything else      -> 500 Internal (detail preserved for logs)
func mapFabricError(err error, resource string) *apperrors.AppError {
	if err == nil {
		return nil
	}
	msg := err.Error()
	low := strings.ToLower(msg)

	switch {
	case strings.Contains(low, "does not exist"), strings.Contains(low, "not found"):
		return apperrors.NewNotFound(resource + " not found")
	case strings.Contains(low, "already exists"):
		return apperrors.NewConflict(resource + " already exists")
	case strings.Contains(low, "not authorized"), strings.Contains(low, "access denied"):
		return apperrors.NewForbidden("not authorized for " + resource)
	case strings.Contains(low, "unauthorized"):
		return apperrors.NewUnauthorized("unauthorized")
	case strings.Contains(low, "invalid "),
		strings.Contains(low, "must be"),
		strings.Contains(low, "is required"),
		strings.Contains(low, "not delivered"),
		strings.Contains(low, "not owned by"),
		strings.Contains(low, "under recall"):
		return apperrors.NewBadRequest(msg)
	default:
		return apperrors.NewInternal("operation failed: "+resource, msg)
	}
}
