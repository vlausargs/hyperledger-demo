package fabric

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-ca/api"
	fabricca "github.com/hyperledger/fabric-ca/lib"
)

// CAClient wraps the fabric-ca client and holds the admin identity.
type CAClient struct {
	caURL       string
	caName      string
	adminMSPDir string
	mspID       string
	homeDir     string
	logger      *slog.Logger
}

// EnrolledIdentity holds the cert/key paths after enrollment.
type EnrolledIdentity struct {
	Name    string `json:"name"`
	MSPID   string `json:"mspID"`
	CertPEM string `json:"certPEM"`
}

// IdentityInfo is a summary of an identity returned by the CA.
type IdentityInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Affiliation string `json:"affiliation"`
	MaxEnroll   int    `json:"maxEnrollments"`
	CAName      string `json:"caName"`
}

// NewCAClient initialises the CA client. adminMSPDir must contain a valid
// enrolled admin identity (signcerts + keystore).
func NewCAClient(caURL, caName, adminMSPDir, mspID, homeDir string) (*CAClient, error) {
	logger := slog.Default().With("component", "ca-client")
	c := &CAClient{
		caURL:       caURL,
		caName:      caName,
		adminMSPDir: adminMSPDir,
		mspID:       mspID,
		homeDir:     homeDir,
		logger:      logger,
	}
	// verify admin identity exists
	if _, err := os.Stat(filepath.Join(adminMSPDir, "signcerts")); err != nil {
		return nil, fmt.Errorf("admin MSP directory missing signcerts: %w", err)
	}
	logger.Info("CA client initialised", "url", caURL, "caName", caName)
	return c, nil
}

// newAdminClient returns a fabric-ca client already loaded with the admin identity.
func (c *CAClient) newAdminClient() (*fabricca.Client, *fabricca.Identity, error) {
	client := &fabricca.Client{
		HomeDir: c.homeDir,
		Config: &fabricca.ClientConfig{
			URL:    c.caURL,
			MSPDir: c.adminMSPDir,
		},
	}
	if err := client.Init(); err != nil {
		return nil, nil, fmt.Errorf("failed to init CA client: %w", err)
	}
	adminID, err := client.LoadMyIdentity()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load admin identity: %w", err)
	}
	return client, adminID, nil
}

// RegisterIdentity registers a new identity with the CA and returns the generated secret.
func (c *CAClient) RegisterIdentity(name, idType, affiliation, secret string, maxEnroll int) (string, error) {
	c.logger.Info("registering identity", "name", name, "type", idType)
	_, adminID, err := c.newAdminClient()
	if err != nil {
		return "", err
	}
	req := &api.RegistrationRequest{
		Name:           name,
		Type:           idType,
		Affiliation:    affiliation,
		Secret:         secret,
		MaxEnrollments: maxEnroll,
		CAName:         c.caName,
	}
	resp, err := adminID.Register(req)
	if err != nil {
		return "", fmt.Errorf("registration failed: %w", err)
	}
	c.logger.Info("identity registered", "name", name)
	return resp.Secret, nil
}

// EnrollIdentity enrolls an identity and stores the cert+key under walletDir/<name>/msp.
// Returns the enrolled cert PEM.
func (c *CAClient) EnrollIdentity(name, secret, walletDir string) (*EnrolledIdentity, error) {
	c.logger.Info("enrolling identity", "name", name)

	mspDir := filepath.Join(walletDir, name, "msp")
	if err := os.MkdirAll(mspDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create msp dir: %w", err)
	}

	client := &fabricca.Client{
		HomeDir: walletDir,
		Config: &fabricca.ClientConfig{
			URL:    c.caURL,
			MSPDir: filepath.Join(name, "msp"),
		},
	}
	if err := client.Init(); err != nil {
		return nil, fmt.Errorf("failed to init CA client: %w", err)
	}

	resp, err := client.Enroll(&api.EnrollmentRequest{
		Name:   name,
		Secret: secret,
		CAName: c.caName,
	})
	if err != nil {
		return nil, fmt.Errorf("enrollment failed: %w", err)
	}

	certPEM := resp.Identity.GetECert().Cert()

	c.logger.Info("identity enrolled", "name", name, "mspDir", mspDir)
	return &EnrolledIdentity{
		Name:    name,
		MSPID:   c.mspID,
		CertPEM: string(certPEM),
	}, nil
}

// GetIdentity returns information about a single identity.
func (c *CAClient) GetIdentity(name string) (*IdentityInfo, error) {
	c.logger.Info("getting identity", "name", name)
	_, adminID, err := c.newAdminClient()
	if err != nil {
		return nil, err
	}
	resp, err := adminID.GetIdentity(name, c.caName)
	if err != nil {
		return nil, fmt.Errorf("get identity failed: %w", err)
	}
	return &IdentityInfo{
		ID:          resp.ID,
		Type:        resp.Type,
		Affiliation: resp.Affiliation,
		MaxEnroll:   resp.MaxEnrollments,
		CAName:      resp.CAName,
	}, nil
}

// ListIdentities returns all identities visible to the admin.
func (c *CAClient) ListIdentities() ([]*IdentityInfo, error) {
	c.logger.Info("listing identities")
	_, adminID, err := c.newAdminClient()
	if err != nil {
		return nil, err
	}

	var identities []*IdentityInfo
	cb := func(decoder *json.Decoder) error {
		id := new(api.IdentityInfo)
		if err := decoder.Decode(id); err != nil {
			return err
		}
		identities = append(identities, &IdentityInfo{
			ID:          id.ID,
			Type:        id.Type,
			Affiliation: id.Affiliation,
			MaxEnroll:   id.MaxEnrollments,
			CAName:      c.caName,
		})
		return nil
	}

	if err := adminID.GetAllIdentities(c.caName, cb); err != nil {
		return nil, fmt.Errorf("list identities failed: %w", err)
	}
	if identities == nil {
		identities = make([]*IdentityInfo, 0)
	}
	return identities, nil
}

// RemoveIdentity removes an identity from the CA and deletes its local wallet
// directory (walletDir/<name>) if it exists. walletDir may be empty to skip
// local cleanup.
func (c *CAClient) RemoveIdentity(name, walletDir string) error {
	c.logger.Info("removing identity", "name", name)
	_, adminID, err := c.newAdminClient()
	if err != nil {
		return err
	}
	_, err = adminID.RemoveIdentity(&api.RemoveIdentityRequest{
		ID:     name,
		CAName: c.caName,
	})
	if err != nil {
		return fmt.Errorf("remove identity failed: %w", err)
	}
	c.logger.Info("identity removed from CA", "name", name)

	if walletDir != "" {
		walletEntry := filepath.Join(walletDir, name)
		if _, statErr := os.Stat(walletEntry); statErr == nil {
			if removeErr := os.RemoveAll(walletEntry); removeErr != nil {
				c.logger.Warn("failed to remove wallet entry", "path", walletEntry, "error", removeErr)
			} else {
				c.logger.Info("wallet entry removed", "path", walletEntry)
			}
		}
	}
	return nil
}

// GetCAURL returns the CA URL.
func (c *CAClient) GetCAURL() string { return c.caURL }

// GetCAName returns the CA name.
func (c *CAClient) GetCAName() string { return c.caName }
