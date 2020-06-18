package config

import (
	"time"
)

type flowExecuteFunc func(f Flow, name string, ex Executor, val interface{},
	tm time.Duration) (err error)

var flowRegistry = make(map[string]flowExecuteFunc)

func register(name string, fn flowExecuteFunc) {
	flowRegistry[name] = fn
}

func init() {

	// common setups

	register("set_monitor", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setMonitor(ex, val, tm)
	})
	register("cleanup_bc", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return ex.CleanupBC(tm)
	})

	// common nodes control (start / stop, lock / unlock)

	register("start", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.start(name, ex, val, false, tm)
	})
	register("start_lock", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.start(name, ex, val, true, tm)
	})
	register("unlock", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.unlock(ex, val, tm)
	})
	register("stop", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.stop(ex, val, tm)
	})

	// wait for an event of the monitor

	register("wait_view_change", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitViewChange(ex, val, tm)
	})
	register("wait_phase", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitPhase(ex, val, tm)
	})
	register("wait_round", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitRound(ex, val, tm)
	})
	register("wait_contribute_mpk", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitContributeMpk(ex, val, tm)
	})
	register("wait_share_signs_or_shares", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitShareSignsOrShares(ex, val, tm)
	})
	register("wait_add", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitAdd(ex, val, tm)
	})
	register("wait_no_progress", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitNoProgress(ex, tm)
	})

	// control nodes behavior / misbehavior (view change)

	register("set_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, true, tm)
	})
	register("unset_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, false, tm)
	})

	// Byzantine blockchain.

	register("vrfs", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vrfs VRFS
		if err = vrfs.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VRFS(&vrfs)
	})

	register("round_timeout", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var rt RoundTimeout
		if err = rt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.RoundTimeout(&rt)
	})

	register("competing_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cb CompetingBlock
		if err = cb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.CompetingBlock(&cb)
	})

	register("sign_only_competing_blocks", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var socb SignOnlyCompetingBlocks
		if err = socb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.SignOnlyCompetingBlocks(&socb)
	})

	register("double_spend_transaction", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var dst DoubleSpendTransaction
		if err = dst.Unmarshal(name, val); err != nil {
			return
		}
		return ex.DoubleSpendTransaction(&dst)
	})

	register("wrong_block_sign_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsh WrongBlockSignHash
		if err = wbsh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignHash(&wbsh)
	})

	register("wrong_block_sign_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsk WrongBlockSignKey
		if err = wbsk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignKey(&wbsk)
	})

	register("wrong_block_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbh WrongBlockHash
		if err = wbh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockHash(&wbh)
	})

	register("verification_ticket", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vt VerificationTicket
		if err = vt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerificationTicket(&vt)
	})

	register("wrong_verification_ticket_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvth WrongVerificationTicketHash
		if err = wvth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketHash(&wvth)
	})

	register("wrong_verification_ticket_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvtk WrongVerificationTicketKey
		if err = wvtk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketKey(&wvtk)
	})

	register("wrong_notarized_block_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnth WrongNotarizedBlockHash
		if err = wnth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockHash(&wnth)
	})

	register("wrong_notarized_block_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnbk WrongNotarizedBlockKey
		if err = wnbk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockKey(&wnbk)
	})

	register("notarize_only_competing_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nocb NotarizeOnlyCompetingBlock
		if err = nocb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizeOnlyCompetingBlock(&nocb)
	})

	register("notarized_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nb NotarizedBlock
		if err = nb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizedBlock(&nb)
	})

	// Byzantine blockchain sharders

	register("finalized_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var fb FinalizedBlock
		if err = fb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.FinalizedBlock(&fb)
	})

	register("magic_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mb MagicBlock
		if err = mb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MagicBlock(&mb)
	})

	register("verify_transaction", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vt VerifyTransaction
		if err = vt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerifyTransaction(&vt)
	})

	register("sc_state", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var scs SCState
		if err = scs.Unmarshal(name, val); err != nil {
			return
		}
		return ex.SCState(&scs)
	})

	// Byzantine view change

	register("mpk", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mpk MPK
		if err = mpk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MPK(&mpk)
	})

	register("shares", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var shares Shares
		if err = shares.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Shares(&shares)
	})

	register("signatures", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var signatures Signatures
		if err = signatures.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Signatures(&signatures)
	})

	register("publish", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var publish Publish
		if err = publish.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Publish(&publish)
	})

}
