package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// NetworkHandler routes channel/chaincode/peer topology endpoints.
type NetworkHandler struct {
	svc *service.NetworkService
}

func NewNetworkHandler(svc *service.NetworkService) *NetworkHandler {
	return &NetworkHandler{svc: svc}
}

func (h *NetworkHandler) Channels(c *gin.Context) {
	response.OK(c, gin.H{
		"channels": []gin.H{
			{"channel_id": h.svc.ChannelID(), "status": "active"},
		},
	})
}

func (h *NetworkHandler) ChannelInfo(c *gin.Context) {
	channelID := c.Param("channelId")
	if vErr := validateID("channel", channelID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	if channelID != h.svc.ChannelID() {
		response.NotFound(c, "Channel not found")
		return
	}
	response.OK(c, gin.H{
		"channel_id": channelID,
		"status":     "active",
		"chaincode":  h.svc.ChaincodeID(),
	})
}

func (h *NetworkHandler) Chaincodes(c *gin.Context) {
	response.OK(c, gin.H{
		"chaincodes": []gin.H{
			{
				"name":    h.svc.ChaincodeID(),
				"version": "3.0",
				"channel": h.svc.ChannelID(),
				"status":  "active",
			},
		},
	})
}

func (h *NetworkHandler) ChaincodeInfo(c *gin.Context) {
	chaincodeID := c.Param("chaincodeId")
	if vErr := validateID("chaincode", chaincodeID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	if chaincodeID != h.svc.ChaincodeID() {
		response.NotFound(c, "Chaincode not found")
		return
	}
	response.OK(c, gin.H{
		"name":     chaincodeID,
		"version":  "3.0",
		"channel":  h.svc.ChannelID(),
		"status":   "active",
		"language": "golang",
		"domain":   "supply-chain",
	})
}

func (h *NetworkHandler) Peers(c *gin.Context) {
	cp, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	var peers []gin.H
	for name, peer := range cp.Peers {
		peers = append(peers, gin.H{"name": name, "address": peer.URL})
	}
	response.OK(c, gin.H{"peers": peers})
}

func (h *NetworkHandler) Organizations(c *gin.Context) {
	cp, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	var orgs []gin.H
	for name, org := range cp.Organizations {
		orgs = append(orgs, gin.H{"name": name, "mspid": org.MSPID, "peers": org.Peers})
	}
	response.OK(c, gin.H{"organizations": orgs})
}

func (h *NetworkHandler) ConnectionProfile(c *gin.Context) {
	cp, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	orgName := cp.Client.Organization
	org, orgExists := cp.Organizations[orgName]

	resp := gin.H{
		"profile": gin.H{
			"name":    cp.Name,
			"version": cp.Version,
		},
		"client": gin.H{
			"organization": cp.Client.Organization,
		},
		"organizations": cp.Organizations,
		"peers": gin.H{
			"count": len(cp.Peers),
			"list":  peerInfo(cp),
		},
		"certificateAuthorities": gin.H{
			"count": len(cp.CAs),
			"list":  caInfo(cp),
		},
	}
	if orgExists {
		resp["currentOrganization"] = gin.H{
			"name":  orgName,
			"mspid": org.MSPID,
			"peers": org.Peers,
			"cas":   org.CertificateAuthorities,
		}
	}
	response.OK(c, resp)
}

func (h *NetworkHandler) Transactions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction history query not supported by Fabric Gateway SDK"})
}

func (h *NetworkHandler) Transaction(c *gin.Context) {
	txID := c.Param("txId")
	if vErr := validateID("transaction", txID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction lookup not supported by Fabric Gateway SDK"})
}

// peerInfo extracts peer information from connection profile (used by
// /network/connection-profile only).
func peerInfo(cp *fabric.ConnectionProfile) []gin.H {
	var peers []gin.H
	for name, peer := range cp.Peers {
		p := gin.H{
			"name": name,
			"url":  peer.URL,
		}
		if peer.TLSCACerts.Pem != "" {
			p["hasTLSCert"] = true
			p["tlsCertLength"] = len(peer.TLSCACerts.Pem)
		} else {
			p["hasTLSCert"] = false
		}
		if peer.GRPCOptions != nil {
			p["grpcOptions"] = peer.GRPCOptions
		}
		peers = append(peers, p)
	}
	return peers
}

// caInfo extracts CA information from connection profile.
func caInfo(cp *fabric.ConnectionProfile) []gin.H {
	var cas []gin.H
	for name, ca := range cp.CAs {
		info := gin.H{
			"name":   name,
			"url":    ca.URL,
			"caName": ca.CAName,
		}
		if ca.TLSCACerts.Pem != "" {
			info["hasTLSCert"] = true
			info["tlsCertLength"] = len(ca.TLSCACerts.Pem)
		} else {
			info["hasTLSCert"] = false
		}
		cas = append(cas, info)
	}
	return cas
}
