package ledger

import (
	"crypto/x509"
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// Minimal in-memory stub (implements shim.ChaincodeStubInterface)
// ---------------------------------------------------------------------------

type mockStub struct {
	state map[string][]byte
}

func newMockStub() *mockStub {
	return &mockStub{state: make(map[string][]byte)}
}

func (m *mockStub) GetState(key string) ([]byte, error)      { return m.state[key], nil }
func (m *mockStub) PutState(key string, value []byte) error  { m.state[key] = value; return nil }
func (m *mockStub) DelState(key string) error                 { delete(m.state, key); return nil }

// ---- Unused stubs to satisfy the interface ----

func (m *mockStub) GetArgs() [][]byte                                           { return nil }
func (m *mockStub) GetStringArgs() []string                                      { return nil }
func (m *mockStub) GetFunctionAndParameters() (string, []string)                 { return "", nil }
func (m *mockStub) GetArgsSlice() ([]byte, error)                                { return nil, nil }
func (m *mockStub) GetTxID() string                                              { return "mock-tx-id" }
func (m *mockStub) GetChannelID() string                                         { return "mock-channel" }
func (m *mockStub) InvokeChaincode(name string, args [][]byte, channel string) *peer.Response {
	return nil
}
func (m *mockStub) SetStateValidationParameter(key string, ep []byte) error      { return nil }
func (m *mockStub) GetStateValidationParameter(key string) ([]byte, error)       { return nil, nil }
func (m *mockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &emptyStateIter{}, &peer.QueryResponseMetadata{Bookmark: ""}, nil
}
func (m *mockStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &emptyStateIter{}, &peer.QueryResponseMetadata{Bookmark: ""}, nil
}
func (m *mockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return objectType, nil
}
func (m *mockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return compositeKey, nil, nil
}
func (m *mockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetQueryResultWithPagination(query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &emptyStateIter{}, &peer.QueryResponseMetadata{Bookmark: ""}, nil
}
func (m *mockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return &emptyHistoryIter{}, nil
}
func (m *mockStub) GetPrivateData(collection, key string) ([]byte, error)             { return nil, nil }
func (m *mockStub) GetPrivateDataHash(collection, key string) ([]byte, error)         { return nil, nil }
func (m *mockStub) PutPrivateData(collection, key string, value []byte) error         { return nil }
func (m *mockStub) DelPrivateData(collection, key string) error                       { return nil }
func (m *mockStub) PurgePrivateData(collection, key string) error                     { return nil }
func (m *mockStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	return nil
}
func (m *mockStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	return nil, nil
}
func (m *mockStub) GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	return &emptyStateIter{}, nil
}
func (m *mockStub) GetCreator() ([]byte, error)                        { return nil, nil }
func (m *mockStub) GetTransient() (map[string][]byte, error)           { return nil, nil }
func (m *mockStub) GetBinding() ([]byte, error)                        { return nil, nil }
func (m *mockStub) GetDecorations() map[string][]byte                  { return nil }
func (m *mockStub) GetSignedProposal() (*peer.SignedProposal, error)   { return nil, nil }
func (m *mockStub) GetTxTimestamp() (*timestamppb.Timestamp, error)    { return timestamppb.Now(), nil }
func (m *mockStub) SetEvent(name string, payload []byte) error         { return nil }

// emptyStateIter — always empty iterator for StateQueryIteratorInterface
type emptyStateIter struct{}

func (e *emptyStateIter) HasNext() bool              { return false }
func (e *emptyStateIter) Close() error               { return nil }
func (e *emptyStateIter) Next() (*queryresult.KV, error) { return nil, nil }

// emptyHistoryIter — always empty iterator for HistoryQueryIteratorInterface
type emptyHistoryIter struct{}

func (e *emptyHistoryIter) HasNext() bool                            { return false }
func (e *emptyHistoryIter) Close() error                             { return nil }
func (e *emptyHistoryIter) Next() (*queryresult.KeyModification, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Minimal mock ClientIdentity
// ---------------------------------------------------------------------------

type mockClientIdentity struct {
	mspID string
}

func (m *mockClientIdentity) GetID() (string, error)   { return "mock-id", nil }
func (m *mockClientIdentity) GetMSPID() (string, error) { return m.mspID, nil }
func (m *mockClientIdentity) GetAttributeValue(attrName string) (string, bool, error) {
	return "", false, nil
}
func (m *mockClientIdentity) AssertAttributeValue(attrName, attrValue string) error { return nil }
func (m *mockClientIdentity) GetX509Certificate() (*x509.Certificate, error)        { return nil, nil }

// Ensure mockClientIdentity satisfies cid.ClientIdentity
var _ cid.ClientIdentity = (*mockClientIdentity)(nil)

// ---------------------------------------------------------------------------
// Minimal mock TransactionContext
// ---------------------------------------------------------------------------

type mockCtx struct {
	stub     shim.ChaincodeStubInterface
	identity cid.ClientIdentity
}

func (c *mockCtx) GetStub() shim.ChaincodeStubInterface { return c.stub }
func (c *mockCtx) GetClientIdentity() cid.ClientIdentity { return c.identity }

func newMockCtx() *mockCtx {
	return &mockCtx{
		stub:     newMockStub(),
		identity: &mockClientIdentity{mspID: "TestMSP"},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

type sampleAsset struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestPutJSONAndGetByID(t *testing.T) {
	store := NewStore()
	ctx := newMockCtx()

	asset := sampleAsset{ID: "asset1", Name: "widget", Value: 42}

	if err := store.PutJSON(ctx, "asset1", asset); err != nil {
		t.Fatalf("PutJSON failed: %v", err)
	}

	var got sampleAsset
	if err := store.GetByID(ctx, "asset1", &got); err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != asset.ID || got.Name != asset.Name || got.Value != asset.Value {
		t.Errorf("GetByID returned %+v, want %+v", got, asset)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store := NewStore()
	ctx := newMockCtx()

	var got sampleAsset
	err := store.GetByID(ctx, "nonexistent", &got)
	if err == nil {
		t.Fatal("expected error for nonexistent key, got nil")
	}
}

func TestExists(t *testing.T) {
	store := NewStore()
	ctx := newMockCtx()

	exists, err := store.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected false for missing key, got true")
	}

	// Put a value and check again
	data, _ := json.Marshal(sampleAsset{ID: "key1"})
	if err := ctx.GetStub().PutState("key1", data); err != nil {
		t.Fatalf("PutState failed: %v", err)
	}

	exists, err = store.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected true for existing key, got false")
	}
}
