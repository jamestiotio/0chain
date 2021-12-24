package stats

import (
	"encoding/json"
	"sync"
)

type (
	// NodesServerStats represents struct with maps containing
	// needed nodes server stats.
	NodesServerStats struct {
		blockMu sync.Mutex

		// Block represents map or storing fetching block stats.
		// minerID -> BlockRequests
		Block map[string]*BlockRequests
	}

	BlockRequests struct {
		listMu sync.Mutex
		list   []*BlockRequest
	}

	// BlockRequest represents struct for collecting reports from the nodes
	// about handled block's requests.
	BlockRequest struct {
		NodeID  string `json:"miner_id"`
		Hash    string `json:"hash"`
		Round   int    `json:"round"`
		Handler string `json:"path"`

		// optional field
		SenderID string `json:"sender_id,omitempty"`
	}
)

// NewNodeServerStats creates initialised NodesServerStats.
func NewNodeServerStats() *NodesServerStats {
	return &NodesServerStats{
		Block: make(map[string]*BlockRequests),
	}
}

// AddBlockStats takes needed info from the BlockRequest and inserts it to the NodesServerStats.Block map.
func (nss *NodesServerStats) AddBlockStats(rep *BlockRequest) {
	nss.blockMu.Lock()
	defer nss.blockMu.Unlock()

	_, ok := nss.Block[rep.NodeID]
	if !ok {
		nss.Block[rep.NodeID] = NewBlockRequests()
	}
	nss.Block[rep.NodeID].Add(rep)
}

// NewBlockRequests creates initialised BlockRequests.
func NewBlockRequests() *BlockRequests {
	return &BlockRequests{
		list: make([]*BlockRequest, 0),
	}
}

// Add adds BlockRequest to the list.
func (bi *BlockRequests) Add(rep *BlockRequest) {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	bi.list = append(bi.list, rep)
}

// GetByHashOrRound looks for BlockRequest with provided hash or round or hash and round both.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetByHashOrRound(hash string, round int) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		onlyHash := stats.Hash == hash && stats.Round == 0
		onlyRound := stats.Round == round && stats.Hash == ""
		hashAndRound := stats.Hash == hash && stats.Round == round
		if onlyHash || onlyRound || hashAndRound {
			return stats
		}
	}
	return nil
}

// GetByHash looks for BlockRequest with provided hash.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetByHash(hash string) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		if stats.Hash == hash {
			return stats
		}
	}
	return nil
}

// GetBySenderIDAndHash looks for BlockRequest with provided senderID and hash.
// Returns nil if BlockRequest was not found.
func (bi *BlockRequests) GetBySenderIDAndHash(senderID, hash string) *BlockRequest {
	bi.listMu.Lock()
	defer bi.listMu.Unlock()

	for _, stats := range bi.list {
		if stats.SenderID == senderID && stats.Hash == hash {
			return stats
		}
	}
	return nil
}

// Encode encodes BlockRequest to the bytes.
func (br *BlockRequest) Encode() ([]byte, error) {
	return json.Marshal(br)
}

// Decode decodes BlockRequest from the bytes.
func (br *BlockRequest) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}
