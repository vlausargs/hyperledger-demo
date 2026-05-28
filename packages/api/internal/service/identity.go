package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// RegisterIdentityInput is the typed payload for IdentityService.Register.
type RegisterIdentityInput struct {
	Name        string
	Type        string
	Affiliation string
	Secret      string
	MaxEnroll   int
}

// EnrollIdentityInput is the typed payload for IdentityService.Enroll.
type EnrollIdentityInput struct {
	Name   string
	Secret string
	Force  bool
}

// RegisterResult is returned by Register.
type RegisterResult struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
	Type   string `json:"type"`
}

// EnrollResult is returned by Enroll.
type EnrollResult struct {
	Name       string `json:"name"`
	MSPID      string `json:"mspID"`
	CertPEM    string `json:"certPEM"`
	WalletPath string `json:"walletPath"`
}

// IdentityService wraps the fabric CA admin client. If ca is nil the
// service still exists but every method returns ErrUnavailable.
type IdentityService struct {
	ca         *fabric.CAClient
	walletPath string
	mspID      string
}

func NewIdentityService(ca *fabric.CAClient, walletPath, mspID string) *IdentityService {
	return &IdentityService{ca: ca, walletPath: walletPath, mspID: mspID}
}

// Available reports whether the CA client was configured at startup.
func (s *IdentityService) Available() bool { return s.ca != nil }

// CAName returns the fabric CA name (used by list response envelopes).
func (s *IdentityService) CAName() string {
	if s.ca == nil {
		return ""
	}
	return s.ca.GetCAName()
}

// List returns every identity visible to the admin.
func (s *IdentityService) List(_ context.Context) ([]*fabric.IdentityInfo, *apperrors.AppError) {
	if s.ca == nil {
		return nil, apperrors.NewUnavailable("identity service not configured")
	}
	out, err := s.ca.ListIdentities()
	if err != nil {
		return nil, apperrors.NewInternal("failed to list identities", err.Error())
	}
	return out, nil
}

// Get returns a single identity by name. Returns ErrNotFound if unknown.
func (s *IdentityService) Get(_ context.Context, name string) (*fabric.IdentityInfo, *apperrors.AppError) {
	if s.ca == nil {
		return nil, apperrors.NewUnavailable("identity service not configured")
	}
	id, err := s.ca.GetIdentity(name)
	if err != nil {
		return nil, apperrors.NewNotFound("identity not found")
	}
	return id, nil
}

// Register creates a new CA identity, returning the generated secret.
func (s *IdentityService) Register(_ context.Context, in RegisterIdentityInput) (*RegisterResult, *apperrors.AppError) {
	if s.ca == nil {
		return nil, apperrors.NewUnavailable("identity service not configured")
	}
	if in.Type == "" {
		in.Type = "client"
	}
	secret, err := s.ca.RegisterIdentity(in.Name, in.Type, in.Affiliation, in.Secret, in.MaxEnroll)
	if err != nil {
		return nil, apperrors.NewInternal("registration failed", err.Error())
	}
	return &RegisterResult{Name: in.Name, Secret: secret, Type: in.Type}, nil
}

// ErrAlreadyEnrolled is returned by Enroll when the wallet entry already
// exists and the caller did not pass Force=true.
var ErrAlreadyEnrolled = errors.New("identity already enrolled")

// Enroll enrolls an identity and writes the resulting MSP into the wallet.
// Returns Conflict if the wallet entry exists and !Force.
func (s *IdentityService) Enroll(_ context.Context, in EnrollIdentityInput) (*EnrollResult, *apperrors.AppError) {
	if s.ca == nil {
		return nil, apperrors.NewUnavailable("identity service not configured")
	}
	walletEntry := filepath.Join(s.walletPath, in.Name)
	if _, err := os.Stat(walletEntry); err == nil && !in.Force {
		return nil, apperrors.NewConflict("identity already enrolled (use force=true to re-enroll)")
	}
	enrolled, err := s.ca.EnrollIdentity(in.Name, in.Secret, s.walletPath)
	if err != nil {
		return nil, apperrors.NewInternal("enrollment failed", err.Error())
	}
	return &EnrollResult{
		Name:       enrolled.Name,
		MSPID:      enrolled.MSPID,
		CertPEM:    enrolled.CertPEM,
		WalletPath: filepath.Join(s.walletPath, in.Name, "msp"),
	}, nil
}

// Delete removes an identity from the CA and its wallet entry.
func (s *IdentityService) Delete(_ context.Context, name string) *apperrors.AppError {
	if s.ca == nil {
		return apperrors.NewUnavailable("identity service not configured")
	}
	if err := s.ca.RemoveIdentity(name, s.walletPath); err != nil {
		return apperrors.NewInternal("failed to remove identity", err.Error())
	}
	return nil
}
