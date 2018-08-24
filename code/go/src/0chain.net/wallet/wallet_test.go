package wallet

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"0chain.net/state"

	"0chain.net/util"
)

var debug = false
var randTime = time.Now().UnixNano()

var prng *rand.Rand

const (
	PERSIST = 1
	MEMORY  = 2
	LEVEL   = 3
)

func TestMPTWithWalletTxns(t *testing.T) {
	clients := 1000
	transactions := 10000
	var rs = rand.NewSource(randTime)
	prng = rand.New(rs)
	wallets := createWallets(clients)
	prng = rand.New(rs)
	fmt.Printf("using in no db\n")
	testWithMPT(nil, wallets, transactions)
	prng = rand.New(rs)
	fmt.Printf("using in memory db\n")
	testWithMPT(GetMPT(MEMORY), wallets, transactions)
	prng = rand.New(rs)
	fmt.Printf("using level db\n")
	lmpt := GetMPT(LEVEL)
	testWithMPT(lmpt, wallets, transactions)
	pmpt := GetMPT(PERSIST)
	ts := time.Now()
	lmpt.GetChangeCollector().UpdateChanges(pmpt.GetNodeDB(), util.Origin(2010), false)
	fmt.Printf("time taken to persist: %v\n", time.Since(ts))
	/*
		prng = rand.New(rs)
		fmt.Printf("using persist db\n")
		testWithMPT(pmpt, wallets, transactions)
	*/
}

func GetMPT(dbType int) util.MerklePatriciaTrieI {
	var mpt util.MerklePatriciaTrieI

	switch dbType {
	case MEMORY:
		mndb := util.NewMemoryNodeDB()
		mpt = util.NewMerklePatriciaTrie(mndb)
	case PERSIST:
		pndb, err := util.NewPNodeDB("/tmp/mpt")
		if err != nil {
			panic(err)
		}
		mpt = util.NewMerklePatriciaTrie(pndb)
	case LEVEL:
		mndb := util.NewMemoryNodeDB()
		pndb := util.NewMemoryNodeDB()
		lndb := util.NewLevelNodeDB(mndb, pndb, false)
		mpt = util.NewMerklePatriciaTrie(lndb)
	}
	return mpt
}

func testWithMPT(mpt util.MerklePatriciaTrieI, wallets []*Wallet, transactions int) {
	if debug {
		fmt.Printf("INFO: random source seed %d\n", randTime)
	}
	if mpt != nil {
		for idx, w := range wallets {
			balance := state.Balance(w.Balance)
			mpt.Insert(util.Path(w.ClientID), &state.State{Balance: balance})
			state, err := getState(mpt, w.ClientID)
			if err != nil {
				panic(err)
			}
			if debug {
				fmt.Printf("INFO:(%v) id:%v balance:%v (%v)\n", idx, w.ClientID, w.Balance, state.Balance)
			}
		}
	}
	if mpt != nil {
		fmt.Printf("wallet creation - num changes: %v\n", len(mpt.GetChangeCollector().GetChanges()))
	}
	if transactions == 0 {
		return
	}
	//mpt.ResetChangeCollector(nil)
	ts := time.Now()
	for count := 1; count <= transactions; count++ {
		var wf, wt *Wallet
		csize := len(wallets)
		for true {
			wf = wallets[prng.Intn(csize)]
			if wf.Balance == 0 {
				continue
			}
			wt = wallets[prng.Intn(csize)]
			if wf != wt {
				break
			}
		}

		value := prng.Int63n(wf.Balance) + 1
		wf.Balance -= value
		wt.Balance += value
		if wf.Balance == 0 {
			if debug {
				fmt.Printf("INFO: deleting wallet of %v as balance is zero\n", wf.ClientID)
			}
			if mpt != nil {
				mpt.Delete(util.Path(wf.ClientID))
			}
		} else {
			if debug {
				fmt.Printf("INFO: moving balance %v from %v to %v\n", value, wf.ClientID, wt.ClientID)
			}
			if mpt != nil {
				s, err := getState(mpt, wf.ClientID)
				if err != nil {
					panic(err)
				}
				s.Balance -= state.Balance(value)
				mpt.Insert(util.Path(wf.ClientID), s)
			}
		}
		if mpt != nil {
			s, err := getState(mpt, wt.ClientID)
			if err != nil {
				panic(err)
			}
			s.Balance += state.Balance(value)
			mpt.Insert(util.Path(wt.ClientID), s)
		}
	}
	if mpt != nil {
		fmt.Printf("transactions - num changes: %v\n", len(mpt.GetChangeCollector().GetChanges()))
	}
	fmt.Printf("transactions - time taken: %v\n", time.Since(ts))
	if mpt == nil {
		return
	}
	zbcount := 0
	for index := 0; index < len(wallets); index++ {
		w := wallets[index]
		if w.Balance == 0 {
			zbcount++
		}
		s, err := getState(mpt, w.ClientID)
		if err != nil {
			if err == util.ErrNodeNotFound {
				fmt.Printf("Node not found; client - %s\n", w.ClientID)
			} else if err == util.ErrValueNotPresent {
				fmt.Printf("Client %s - deleted ; (Balance - %d)\n", w.ClientID, w.Balance)
			}
		} else {
			if s.Balance != state.Balance(w.Balance) {
				fmt.Printf("balance mismatch (%v): %d; Found : %d\n", w.ClientID, w.Balance, s.Balance)
			}
		}
	}
	fmt.Printf("zero balance clients %v\n", zbcount)
}

func createWallets(num int) []*Wallet {
	wallets := make([]*Wallet, num)
	for i := 0; i < len(wallets); i++ {
		balance := prng.Int63n(1000)
		wallets[i] = &Wallet{Balance: balance}
		wallets[i].Initialize()
	}
	return wallets
}

func getState(mpt util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	s := &state.State{}
	s.Balance = state.Balance(0)
	ss, err := mpt.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
	} else {
		deserializer := &state.Deserializer{}
		s = deserializer.Deserialize(ss).(*state.State)
	}
	return s, nil
}