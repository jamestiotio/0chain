package miner

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chain"

	"0chain.net/block"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

var InsufficientTxns = "insufficient_txns"

/*StartRound - start a new round */
func (mc *Chain) StartRound(ctx context.Context, r *Round) {
	mc.AddRound(r)
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block, bsh chain.BlockStateHandler) error {
	clients := make(map[string]*client.Client)
	b.Txns = make([]*transaction.Transaction, mc.BlockSize)
	//TODO: wasting this because []interface{} != []*transaction.Transaction in Go
	etxns := make([]datastore.Entity, mc.BlockSize)
	var invalidTxns []datastore.Entity
	var idx int32
	var ierr error
	var count int32
	var roundMismatch bool
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		if mc.CurrentRound > b.Round {
			roundMismatch = true
			return false
		}
		count++
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			return true
		}

		//TODO: this needs to be uncommented once we have a way to generate a steady stream of transactions
		/*
			if !common.Within(int64(txn.CreationDate), 5) {
				invalidTxns = append(invalidTxns, qe)
			}*/

		if ok, err := b.PrevBlock.ChainHasTransaction(txn); ok || err != nil {
			if err != nil {
				ierr = err
			}
			return true
		}

		//Setting the score lower so the next time blocks are generated these transactions don't show up at the top
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)

		b.Txns[idx] = txn
		etxns[idx] = txn
		b.AddTransaction(txn)
		idx++

		clients[txn.ClientID] = nil

		if idx == mc.BlockSize {
			return false
		}
		return true
	}

	start := time.Now()
	b.CreationDate = common.Now()
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	err := transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if roundMismatch {
		Logger.Error("generate block (round mismatch)", zap.Any("round", b.Round), zap.Any("current_round", mc.CurrentRound))
		return common.NewError("round_mismatch", "current round different from generation round")
	}
	if ierr != nil {
		Logger.Error("generate block (txn reinclusion check)", zap.Any("round", b.Round), zap.Error(ierr))
	}
	if len(invalidTxns) > 0 {
		Logger.Error("generate block (found invalid txns)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(invalidTxns)))
		go mc.deleteTxns(invalidTxns)
	}
	if err != nil {
		return err
	}

	if idx != mc.BlockSize {
		b.Txns = nil
		Logger.Debug("generate block (insufficient txns)", zap.Any("round", b.Round), zap.Any("iteration_count", count), zap.Any("block_size", mc.BlockSize), zap.Any("num_txns", idx))
		return common.NewError(InsufficientTxns, fmt.Sprintf("not sufficient txns to make a block yet for round %v", b.Round))
	}
	if count > 10*mc.BlockSize {
		Logger.Debug("generate block (too much iteration", zap.Any("round", b.Round), zap.Any("iteration_count", count))
	}
	client.GetClients(ctx, clients)
	Logger.Debug("generate block (time to assemble block)", zap.Any("round", b.Round), zap.Any("time", time.Since(start)))

	bsh.UpdatePendingBlock(ctx, b, etxns)
	for _, txn := range b.Txns {
		client := clients[txn.ClientID]
		if client == nil {
			Logger.Debug("generate block (invalid client id)", zap.String("client_id", txn.ClientID))
			return common.NewError("invalid_client_id", "client id not available")
		}
		txn.PublicKey = client.PublicKey
		txn.ClientID = datastore.EmptyKey
	}
	Logger.Debug("generate block (time to assemble + update txns)", zap.Any("round", b.Round), zap.Any("time", time.Since(start)))

	self := node.GetSelfNode(ctx)
	b.MinerID = self.ID
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return err
	}
	Logger.Debug("generate block (time to assemble+update+sign block)", zap.Any("round", b.Round), zap.Any("time", time.Since(start)), zap.Any("block", b.Hash))

	go b.ComputeTxnMap()
	return nil
}

/*UpdatePendingBlock - updates the block that is generated and pending rest of the process */
func (mc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")

	//NOTE: Since we are not explicitly maintaining state in the db, we just need to adjust the collection score and don't need to write the entities themselves
	//transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, txns)
	transactionMetadataProvider.GetStore().MultiAddToCollection(ctx, transactionMetadataProvider, txns)
}

/*VerifyBlock - given a set of transaction ids within a block, validate the block */
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	start := time.Now()
	err := b.Validate(ctx)
	if err != nil {
		return nil, err
	}
	hashCameWithBlock := b.Hash
	hash := b.ComputeHash()
	if hashCameWithBlock != hash {
		return nil, common.NewError("hash wrong", "The hash of the block is wrong")
	}
	miner := node.GetNode(b.MinerID)
	if miner == nil {
		return nil, common.NewError("unknown_miner", "Do not know this miner")
	}
	var ok bool
	ok, err = miner.Verify(b.Signature, b.Hash)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, common.NewError("signature invalid", "The block wasn't signed correctly")
	}
	err = b.ValidateTransactions(ctx)
	if err != nil {
		return nil, err
	}
	bvt, err := mc.SignBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	Logger.Debug("block verification time", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("num_txns", len(b.Txns)), zap.Any("duration", time.Since(start)))
	return bvt, nil
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	self := node.GetSelfNode(ctx)
	var err error
	bvt.VerifierID = self.GetKey()
	bvt.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

/*AddVerificationTicket - add a verified ticket to the list of verification tickets of the block */
func (mc *Chain) AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool {
	return b.AddVerificationTicket(bvt)
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	fr := mc.GetRound(b.Round)
	fr.Finalize(b)
	mc.DeleteRound(ctx, &fr.Round)
	mc.FinalizeBlock(ctx, b)
	mc.SendFinalizedBlock(ctx, b)
}

/*FinalizeBlock - finalize the transactions in the block */
func (mc *Chain) FinalizeBlock(ctx context.Context, b *block.Block) error {
	modifiedTxns := make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		modifiedTxns[idx] = txn
	}
	return mc.deleteTxns(modifiedTxns)
}
