package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
)

func GetChannels(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"channels": []gin.H{
				{"channel_id": gw.GetChannel(), "status": "active"},
			},
		})
	}
}

func GetChannelInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("channelId")
		if channelID != gw.GetChannel() {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"channel_id": channelID,
			"status":     "active",
			"chaincode":  gw.GetChaincode(),
		})
	}
}

func GetChaincodes(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"chaincodes": []gin.H{
				{
					"name":    gw.GetChaincode(),
					"version": "3.0",
					"channel": gw.GetChannel(),
					"status":  "active",
				},
			},
		})
	}
}

func GetChaincodeInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		chaincodeID := c.Param("chaincodeId")
		if chaincodeID != gw.GetChaincode() {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chaincode not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"name":     chaincodeID,
			"version":  "3.0",
			"channel":  gw.GetChannel(),
			"status":   "active",
			"language": "golang",
			"domain":   "supply-chain",
		})
	}
}

func GetPeers(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var peers []gin.H
		for name, peer := range cp.Peers {
			peers = append(peers, gin.H{"name": name, "address": peer.URL})
		}
		c.JSON(http.StatusOK, gin.H{"peers": peers})
	}
}

func GetOrganizations(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var orgs []gin.H
		for name, org := range cp.Organizations {
			orgs = append(orgs, gin.H{"name": name, "mspid": org.MSPID, "peers": org.Peers})
		}
		c.JSON(http.StatusOK, gin.H{"organizations": orgs})
	}
}

func GetConnectionProfile(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		connProfile := gw.GetConnectionProfile()
		if connProfile == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Connection profile not loaded",
			})
			return
		}

		orgName := connProfile.Client.Organization
		org, orgExists := connProfile.Organizations[orgName]

		response := gin.H{
			"profile": gin.H{
				"name":    connProfile.Name,
				"version": connProfile.Version,
			},
			"client": gin.H{
				"organization": connProfile.Client.Organization,
			},
			"organizations": connProfile.Organizations,
			"peers": gin.H{
				"count": len(connProfile.Peers),
				"list":  getPeerInfo(connProfile),
			},
			"certificateAuthorities": gin.H{
				"count": len(connProfile.CAs),
				"list":  getCAInfo(connProfile),
			},
		}

		if orgExists {
			response["currentOrganization"] = gin.H{
				"name":  orgName,
				"mspid": org.MSPID,
				"peers": org.Peers,
				"cas":   org.CertificateAuthorities,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

func GetTransactions(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction history query not supported by Fabric Gateway SDK"})
	}
}

func GetTransaction(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction lookup not supported by Fabric Gateway SDK"})
	}
}

// getPeerInfo extracts peer information from connection profile
func getPeerInfo(cp *fabric.ConnectionProfile) []gin.H {
	var peers []gin.H
	for name, peer := range cp.Peers {
		peerInfo := gin.H{
			"name": name,
			"url":  peer.URL,
		}
		if peer.TLSCACerts.Pem != "" {
			peerInfo["hasTLSCert"] = true
			peerInfo["tlsCertLength"] = len(peer.TLSCACerts.Pem)
		} else {
			peerInfo["hasTLSCert"] = false
		}
		if peer.GRPCOptions != nil {
			peerInfo["grpcOptions"] = peer.GRPCOptions
		}
		peers = append(peers, peerInfo)
	}
	return peers
}

// getCAInfo extracts CA information from connection profile
func getCAInfo(cp *fabric.ConnectionProfile) []gin.H {
	var cas []gin.H
	for name, ca := range cp.CAs {
		caInfo := gin.H{
			"name":   name,
			"url":    ca.URL,
			"caName": ca.CAName,
		}
		if ca.TLSCACerts.Pem != "" {
			caInfo["hasTLSCert"] = true
			caInfo["tlsCertLength"] = len(ca.TLSCACerts.Pem)
		} else {
			caInfo["hasTLSCert"] = false
		}
		cas = append(cas, caInfo)
	}
	return cas
}
