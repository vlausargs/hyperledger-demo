package rest

import (
	"log/slog"
	"net/http"
	"os"

	"hlf-demo/fabric-network/client/fabric"

	"github.com/gin-gonic/gin"
)

// RegisterRequest is the body for POST /identities/register.
type RegisterRequest struct {
	Name        string `json:"name"        binding:"required"`
	Type        string `json:"type"`
	Affiliation string `json:"affiliation"`
	Secret      string `json:"secret"`
	MaxEnroll   int    `json:"maxEnrollments"`
}

// EnrollRequest is the body for POST /identities/enroll.
type EnrollRequest struct {
	Name   string `json:"name"   binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// GetIdentities handles GET /identities — list all CA identities.
func GetIdentities(ca *fabric.CAClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		identities, err := ca.ListIdentities()
		if err != nil {
			slog.Error("ListIdentities failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"identities": identities,
			"count":      len(identities),
			"caName":     ca.GetCAName(),
		})
	}
}

// GetIdentity handles GET /identities/:id — get a single CA identity.
func GetIdentity(ca *fabric.CAClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		name := c.Param("id")
		id, err := ca.GetIdentity(name)
		if err != nil {
			slog.Error("GetIdentity failed", "name", name, "error", err, "correlation_id", cid)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, id)
	}
}

// RegisterIdentity handles POST /identities/register — register a new identity.
func RegisterIdentity(ca *fabric.CAClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Type == "" {
			req.Type = "client"
		}
		secret, err := ca.RegisterIdentity(req.Name, req.Type, req.Affiliation, req.Secret, req.MaxEnroll)
		if err != nil {
			slog.Error("RegisterIdentity failed", "name", req.Name, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"name":   req.Name,
			"secret": secret,
			"type":   req.Type,
		})
	}
}

// EnrollIdentity handles POST /identities/enroll — enroll an identity and store cert/key.
// Returns 409 if the identity is already enrolled (wallet entry exists).
// To re-enroll, delete the identity first or call with force=true query param.
func EnrollIdentity(ca *fabric.CAClient, walletPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		var req EnrollRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		walletEntry := walletPath + "/" + req.Name
		if _, err := os.Stat(walletEntry); err == nil && c.Query("force") != "true" {
			c.JSON(http.StatusConflict, gin.H{
				"error":      "identity already enrolled",
				"name":       req.Name,
				"walletPath": walletEntry + "/msp",
				"hint":       "add ?force=true to re-enroll and overwrite existing credentials",
			})
			return
		}

		enrolled, err := ca.EnrollIdentity(req.Name, req.Secret, walletPath)
		if err != nil {
			slog.Error("EnrollIdentity failed", "name", req.Name, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"name":       enrolled.Name,
			"mspID":      enrolled.MSPID,
			"certPEM":    enrolled.CertPEM,
			"walletPath": walletPath + "/" + req.Name + "/msp",
		})
	}
}

// DeleteIdentity handles DELETE /identities/:id — remove an identity from the CA
// and delete its local wallet entry.
func DeleteIdentity(ca *fabric.CAClient, walletPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		name := c.Param("id")
		if err := ca.RemoveIdentity(name, walletPath); err != nil {
			slog.Error("RemoveIdentity failed", "name", name, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "identity removed", "name": name})
	}
}
