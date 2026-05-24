package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) PutJSON(ctx contractapi.TransactionContextInterface, key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", key, err)
	}
	return ctx.GetStub().PutState(key, b)
}

func (s *Store) GetByID(ctx contractapi.TransactionContextInterface, key string, result interface{}) error {
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", key, err)
	}
	if b == nil {
		return fmt.Errorf("%s does not exist", key)
	}
	return json.Unmarshal(b, result)
}

func (s *Store) Exists(ctx contractapi.TransactionContextInterface, key string) (bool, error) {
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", key, err)
	}
	return b != nil, nil
}

func (s *Store) GetByRange(ctx contractapi.TransactionContextInterface, startKey, endKey string, unmarshal func([]byte) error) error {
	iter, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return fmt.Errorf("range query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if err := unmarshal(res.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) QueryBySelector(ctx contractapi.TransactionContextInterface, query string, unmarshal func([]byte) error) error {
	iter, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if err := unmarshal(res.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPaginated(ctx contractapi.TransactionContextInterface, startKey, endKey string, pageSize int32, bookmark string, unmarshal func([]byte) error) (string, error) {
	iter, meta, err := ctx.GetStub().GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark)
	if err != nil {
		return "", fmt.Errorf("paginated query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return "", err
		}
		if err := unmarshal(res.Value); err != nil {
			return "", err
		}
	}
	return meta.Bookmark, nil
}

func (s *Store) GetHistory(ctx contractapi.TransactionContextInterface, key string) ([]HistoryEntry, error) {
	iter, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for %s: %w", key, err)
	}
	defer iter.Close()

	var results []HistoryEntry
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var ts time.Time
		if response.Timestamp != nil {
			ts = response.Timestamp.AsTime()
		}
		var val interface{}
		if err := json.Unmarshal(response.Value, &val); err != nil {
			val = string(response.Value)
		}
		results = append(results, HistoryEntry{
			TxID:      response.TxId,
			Timestamp: ts,
			IsDelete:  response.IsDelete,
			Value:     val,
		})
	}
	return results, nil
}

func CallerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	msp, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get caller MSPID: %w", err)
	}
	return msp, nil
}

func RequireMSP(ctx contractapi.TransactionContextInterface, allowed ...string) (string, error) {
	caller, err := CallerMSPID(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range allowed {
		if caller == a {
			return caller, nil
		}
	}
	return "", fmt.Errorf("caller MSP %s is not authorized; allowed: %v", caller, allowed)
}

func TxTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get tx timestamp: %w", err)
	}
	return ts.AsTime().UTC().Format(time.RFC3339), nil
}

type HistoryEntry struct {
	TxID      string      `json:"txId"`
	Timestamp time.Time   `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     interface{} `json:"value"`
}
