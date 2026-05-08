/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes the basic details of an asset
type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
	OwnerMSP       string `json:"ownerMSP"`
}

// HistoryQueryResult describes a record in a history query
type HistoryQueryResult struct {
	Record    *Asset    `json:"record"`
	TxId      string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
}

// PagedQueryResult wraps paginated asset results with a bookmark
type PagedQueryResult struct {
	Assets   []*Asset `json:"assets"`
	Bookmark string   `json:"bookmark"`
}

// callerMSPID returns the MSP ID of the transaction submitter.
func callerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	msp, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get caller MSPID: %w", err)
	}
	return msp, nil
}

// checkOwner returns an error if the caller's MSP does not own the asset.
// Assets with empty OwnerMSP (pre-access-control) are allowed for all callers.
func checkOwner(ctx contractapi.TransactionContextInterface, asset *Asset) error {
	if asset.OwnerMSP == "" {
		return nil
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != asset.OwnerMSP {
		return fmt.Errorf("caller MSP %s is not the owner of asset %s (owner: %s)", caller, asset.ID, asset.OwnerMSP)
	}
	return nil
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	msp, err := callerMSPID(ctx)
	if err != nil {
		return err
	}

	assets := []Asset{
		{ID: "asset1", Color: "blue", Size: 5, Owner: "Tomoko", AppraisedValue: 300, OwnerMSP: msp},
		{ID: "asset2", Color: "red", Size: 5, Owner: "Brad", AppraisedValue: 400, OwnerMSP: msp},
		{ID: "asset3", Color: "green", Size: 10, Owner: "Jin Soo", AppraisedValue: 500, OwnerMSP: msp},
		{ID: "asset4", Color: "yellow", Size: 10, Owner: "Max", AppraisedValue: 600, OwnerMSP: msp},
		{ID: "asset5", Color: "black", Size: 15, Owner: "Adriana", AppraisedValue: 700, OwnerMSP: msp},
		{ID: "asset6", Color: "white", Size: 15, Owner: "Michel", AppraisedValue: 800, OwnerMSP: msp},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}
		if err := ctx.GetStub().PutState(asset.ID, assetJSON); err != nil {
			return fmt.Errorf("failed to put asset %s to world state: %w", asset.ID, err)
		}
	}

	return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	msp, err := callerMSPID(ctx)
	if err != nil {
		return err
	}

	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
		OwnerMSP:       msp,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %w", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	if err := json.Unmarshal(assetJSON, &asset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset: %w", err)
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
// Only the asset owner's org may update.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}
	if err := checkOwner(ctx, asset); err != nil {
		return err
	}

	asset.Color = color
	asset.Size = size
	asset.Owner = owner
	asset.AppraisedValue = appraisedValue

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an asset from the world state.
// Only the asset owner's org may delete.
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}
	if err := checkOwner(ctx, asset); err != nil {
		return err
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %w", err)
	}
	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state.
// Only the asset owner's org may transfer.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string) error {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}
	if err := checkOwner(ctx, asset); err != nil {
		return err
	}

	asset.Owner = newOwner
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// GetAllAssets returns all assets found in world state (no pagination).
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get state by range: %w", err)
	}
	defer resultsIterator.Close()

	assets := make([]*Asset, 0)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate assets: %w", err)
		}
		var asset Asset
		if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAllAssetsPaged returns a page of assets with cursor-based pagination.
// pageSize max is 100; bookmark is the cursor from the previous response.
func (s *SmartContract) GetAllAssetsPaged(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*PagedQueryResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	resultsIterator, metadata, err := ctx.GetStub().GetStateByRangeWithPagination("", "", int32(pageSize), bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to get paged state by range: %w", err)
	}
	defer resultsIterator.Close()

	assets := make([]*Asset, 0)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate paged assets: %w", err)
		}
		var asset Asset
		if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return &PagedQueryResult{
		Assets:   assets,
		Bookmark: metadata.Bookmark,
	}, nil
}

// GetAssetHistory returns the history of an asset
func (s *SmartContract) GetAssetHistory(ctx contractapi.TransactionContextInterface, id string) ([]*HistoryQueryResult, error) {
	historyIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for key %s: %w", id, err)
	}
	defer historyIterator.Close()

	results := make([]*HistoryQueryResult, 0)
	for historyIterator.HasNext() {
		response, err := historyIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate history: %w", err)
		}

		var asset Asset
		if len(response.Value) > 0 {
			if err := json.Unmarshal(response.Value, &asset); err != nil {
				return nil, fmt.Errorf("failed to unmarshal asset history record: %w", err)
			}
		}

		var timestamp time.Time
		if response.Timestamp != nil {
			timestamp = response.Timestamp.AsTime()
		}
		results = append(results, &HistoryQueryResult{
			Record:    &asset,
			TxId:      response.TxId,
			Timestamp: timestamp,
			IsDelete:  response.IsDelete,
		})
	}

	return results, nil
}

// GetAssetByRange returns assets in a specified ID range
func (s *SmartContract) GetAssetByRange(ctx contractapi.TransactionContextInterface, startKey string, endKey string) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get state by range [%s, %s]: %w", startKey, endKey, err)
	}
	defer resultsIterator.Close()

	assets := make([]*Asset, 0)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate assets: %w", err)
		}
		var asset Asset
		if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating asset chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting asset chaincode: %v", err)
	}
}
