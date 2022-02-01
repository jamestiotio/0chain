//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"errors"
	"log"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	var state = crpc.Client().State()

	if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		return nil, errors.New("skip block signing -- not competing block")
	}

	// regular or competing signing
	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var (
		self  = node.Self
		state = crpc.Client().State()
	)
	b.HashBlock()

	switch {
	case state.WrongBlockHash != nil:
		b.Hash = revertString(b.Hash) // just wrong block hash
		b.Signature, err = self.Sign(b.Hash)
	case state.WrongBlockSignHash != nil:
		b.Signature, err = self.Sign(revertString(b.Hash)) // sign another hash
	case state.WrongBlockSignKey != nil:
		b.Signature, err = crpcutils.Sign(b.Hash) // wrong secret key
	default:
		b.Signature, err = self.Sign(b.Hash)
	}

	return
}

// has double-spend transaction
func hasDST(pb, b []*transaction.Transaction) (has bool) {
	for _, bx := range b {
		if bx == nil {
			continue
		}
		for _, pbx := range pb {
			if pbx == nil {
				continue
			}
			if bx.Hash == pbx.Hash {
				return true // has
			}
		}
	}
	return false // has not
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	mc.updateFinalizedBlock(ctx, b)

	if isTestingOnUpdateFinalizedBlock(b.Round) {
		if err := addRoundInfoResult(b.Hash, mc.GetRound(b.Round)); err != nil {
			log.Panicf("Conductor: error while sending round info result: %v", err)
		}
	}
}

func isTestingOnUpdateFinalizedBlock(round int64) bool {
	s := crpc.Client().State()
	var isTestingFunc func(round int64, generator bool, typeRank int) bool
	switch {
	case s.ExtendNotNotarisedBlock != nil:
		isTestingFunc = s.ExtendNotNotarisedBlock.IsTesting

	case s.BreakingSingleBlock != nil:
		isTestingFunc = s.BreakingSingleBlock.IsTesting

	case s.SendInsufficientProposals != nil:
		isTestingFunc = s.SendInsufficientProposals.IsTesting

	case s.NotarisingNonExistentBlock != nil:
		isTestingFunc = s.NotarisingNonExistentBlock.IsTesting

	case s.ResendProposedBlock != nil:
		isTestingFunc = s.ResendProposedBlock.IsTesting

	case s.ResendNotarisation != nil:
		isTestingFunc = s.ResendNotarisation.IsTesting

	case s.BadTimeoutVRFS != nil:
		isTestingFunc = s.BadTimeoutVRFS.IsTesting

	case s.BlockStateChangeRequestor != nil:
		isTestingFunc = s.BlockStateChangeRequestor.IsTesting

	case s.MinerNotarisedBlockRequestor != nil:
		isTestingFunc = s.MinerNotarisedBlockRequestor.IsTesting

	default:
		return false
	}

	nodeType, typeRank := getNodeTypeAndTypeRank(round)
	return isTestingFunc(round, nodeType == generator, typeRank)
}

func addRoundInfoResult(finalisedBlockHash string, r round.RoundI) error {
	res := roundInfo(r.GetRoundNumber(), finalisedBlockHash)
	blob, err := res.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
}

func roundInfo(round int64, finalisedBlockHash string) *cases.RoundInfo {
	mc := GetMinerChain()

	miners := mc.GetMiners(round).CopyNodes()
	rankedMiners := make([]string, len(miners))
	roundI := mc.GetRound(round)
	for _, miner := range miners {
		rankedMiners[roundI.GetMinerRank(miner)] = miner.ID
	}

	propBlocks := roundI.GetProposedBlocks()
	propBlocksInfo := make([]*cases.BlockInfo, 0, len(propBlocks))
	for _, b := range propBlocks {
		propBlocksInfo = append(propBlocksInfo, getBlockInfo(b))
	}
	notBlocks := roundI.GetNotarizedBlocks()
	notBlocksInfo := make([]*cases.BlockInfo, 0, len(notBlocks))
	for _, b := range notBlocks {
		notBlocksInfo = append(notBlocksInfo, getBlockInfo(b))
	}

	return &cases.RoundInfo{
		Num:                round,
		GeneratorsNum:      mc.GetGeneratorsNum(),
		RankedMiners:       rankedMiners,
		FinalisedBlockHash: finalisedBlockHash,
		ProposedBlocks:     propBlocksInfo,
		NotarisedBlocks:    notBlocksInfo,
		TimeoutCount:       roundI.GetTimeoutCount(),
		RoundRandomSeed:    roundI.GetRandomSeed(),
		IsFinalised:        roundI.IsFinalized(),
	}
}

func getBlockInfo(b *block.Block) *cases.BlockInfo {
	return &cases.BlockInfo{
		Hash:                b.Hash,
		PrevHash:            b.PrevHash,
		Notarised:           b.IsBlockNotarized(),
		VerificationStatus:  b.GetVerificationStatus(),
		Rank:                b.RoundRank,
		VerificationTickets: getVerificationTicketsInfo(b.VerificationTickets),
	}
}

func getVerificationTicketsInfo(tickets []*block.VerificationTicket) []*cases.VerificationTicketInfo {
	tickInfo := make([]*cases.VerificationTicketInfo, 0, len(tickets))
	for _, ticket := range tickets {
		tickInfo = append(tickInfo, &cases.VerificationTicketInfo{
			VerifierID: ticket.VerifierID,
		})
	}
	return tickInfo
}
