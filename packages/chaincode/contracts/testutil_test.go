package contracts

import (
	"crypto/x509"
	"sort"

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

func (m *mockStub) GetState(key string) ([]byte, error)     { return m.state[key], nil }
func (m *mockStub) PutState(key string, value []byte) error { m.state[key] = value; return nil }
func (m *mockStub) DelState(key string) error               { delete(m.state, key); return nil }

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

// keysInRange returns sorted keys from the state map that fall within [startKey, endKey).
// If endKey is empty, all keys >= startKey are returned.
func (m *mockStub) keysInRange(startKey, endKey string) []string {
	var keys []string
	for k := range m.state {
		if startKey != "" && k < startKey {
			continue
		}
		if endKey != "" && k >= endKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (m *mockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	keys := m.keysInRange(startKey, endKey)
	var items []*queryresult.KV
	for _, k := range keys {
		items = append(items, &queryresult.KV{Key: k, Value: m.state[k]})
	}
	return &sliceStateIter{items: items, idx: -1}, nil
}

func (m *mockStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	keys := m.keysInRange(startKey, endKey)

	// Apply bookmark: skip keys until we pass the bookmark key.
	if bookmark != "" {
		idx := -1
		for i, k := range keys {
			if k == bookmark {
				idx = i
				break
			}
		}
		if idx >= 0 {
			keys = keys[idx+1:]
		}
	}

	// Apply page size limit.
	nextBookmark := ""
	if int(pageSize) > 0 && len(keys) > int(pageSize) {
		nextBookmark = keys[pageSize]
		keys = keys[:pageSize]
	}

	var items []*queryresult.KV
	for _, k := range keys {
		items = append(items, &queryresult.KV{Key: k, Value: m.state[k]})
	}
	return &sliceStateIter{items: items, idx: -1}, &peer.QueryResponseMetadata{Bookmark: nextBookmark}, nil
}

func (m *mockStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return &sliceStateIter{idx: -1}, nil
}
func (m *mockStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &sliceStateIter{idx: -1}, &peer.QueryResponseMetadata{Bookmark: ""}, nil
}
func (m *mockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return objectType, nil
}
func (m *mockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return compositeKey, nil, nil
}
func (m *mockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	return &sliceStateIter{idx: -1}, nil
}
func (m *mockStub) GetQueryResultWithPagination(query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *peer.QueryResponseMetadata, error) {
	return &sliceStateIter{idx: -1}, &peer.QueryResponseMetadata{Bookmark: ""}, nil
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
	return &sliceStateIter{idx: -1}, nil
}
func (m *mockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return &sliceStateIter{idx: -1}, nil
}
func (m *mockStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	return &sliceStateIter{idx: -1}, nil
}
func (m *mockStub) GetCreator() ([]byte, error)                       { return nil, nil }
func (m *mockStub) GetTransient() (map[string][]byte, error)          { return nil, nil }
func (m *mockStub) GetBinding() ([]byte, error)                       { return nil, nil }
func (m *mockStub) GetDecorations() map[string][]byte                 { return nil }
func (m *mockStub) GetSignedProposal() (*peer.SignedProposal, error)  { return nil, nil }
func (m *mockStub) GetTxTimestamp() (*timestamppb.Timestamp, error)   { return timestamppb.Now(), nil }
func (m *mockStub) SetEvent(name string, payload []byte) error        { return nil }

// ---------------------------------------------------------------------------
// State iterator backed by a slice of KV pairs
// ---------------------------------------------------------------------------

type sliceStateIter struct {
	items []*queryresult.KV
	idx   int
}

func (s *sliceStateIter) HasNext() bool {
	return s.idx+1 < len(s.items)
}

func (s *sliceStateIter) Close() error { return nil }

func (s *sliceStateIter) Next() (*queryresult.KV, error) {
	s.idx++
	if s.idx >= len(s.items) {
		return nil, nil
	}
	return s.items[s.idx], nil
}

// ---------------------------------------------------------------------------
// Empty history iterator
// ---------------------------------------------------------------------------

type emptyHistoryIter struct{}

func (e *emptyHistoryIter) HasNext() bool                                    { return false }
func (e *emptyHistoryIter) Close() error                                     { return nil }
func (e *emptyHistoryIter) Next() (*queryresult.KeyModification, error)      { return nil, nil }

// ---------------------------------------------------------------------------
// Mock ClientIdentity
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

var _ cid.ClientIdentity = (*mockClientIdentity)(nil)

// ---------------------------------------------------------------------------
// Mock TransactionContext
// ---------------------------------------------------------------------------

type mockCtx struct {
	stub     shim.ChaincodeStubInterface
	identity cid.ClientIdentity
}

func (c *mockCtx) GetStub() shim.ChaincodeStubInterface { return c.stub }
func (c *mockCtx) GetClientIdentity() cid.ClientIdentity { return c.identity }

func newMockCtx(mspID string) *mockCtx {
	return &mockCtx{
		stub:     newMockStub(),
		identity: &mockClientIdentity{mspID: mspID},
	}
}
