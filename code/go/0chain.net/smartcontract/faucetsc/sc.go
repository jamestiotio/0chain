package faucetsc

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/smartcontract"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	sc "0chain.net/smartcontract"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	name    = "faucet"
)

type FaucetSmartContract struct {
	*smartcontractinterface.SmartContract
}

func NewFaucetSmartContract() smartcontractinterface.SmartContractInterface {
	var fcCopy = &FaucetSmartContract{
		SmartContract: smartcontractinterface.NewSC(ADDRESS),
	}
	fcCopy.setSC(fcCopy.SmartContract, &smartcontract.BCContext{})
	return fcCopy
}

func (ipsc *FaucetSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *FaucetSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (fc *FaucetSmartContract) GetName() string {
	return name
}

func (fc *FaucetSmartContract) GetAddress() string {
	return ADDRESS
}

func (fc *FaucetSmartContract) GetCost(t *transaction.Transaction, funcName string, balances c_state.StateContextI) (int, error) {
	node, err := fc.getGlobalVariables(t, balances)
	if err != nil {
		return math.MaxInt32, err
	}
	if node.Cost == nil {
		return math.MaxInt32, err
	}
	cost, ok := node.Cost[funcName]
	if !ok {
		return math.MaxInt32, err
	}
	return cost, nil
}

func (fc *FaucetSmartContract) setSC(sc *smartcontractinterface.SmartContract, _ smartcontractinterface.BCContextI) {
	fc.SmartContract = sc
	fc.SmartContractExecutionStats["update-settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "update-settings"), nil)
	fc.SmartContractExecutionStats["pour"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "pour"), nil)
	fc.SmartContractExecutionStats["refill"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "refill"), nil)
	fc.SmartContractExecutionStats["tokens Poured"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fc.ID, "tokens Poured"), nil, metrics.NewUniformSample(1024))
	fc.SmartContractExecutionStats["token refills"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fc.ID, "token refills"), nil, metrics.NewUniformSample(1024))
}

func (un *UserNode) validPourRequest(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (bool, error) {
	smartContractBalance, err := balances.GetClientBalance(gn.ID)
	if err == util.ErrValueNotPresent {
		return false, common.NewError("invalid_request", "faucet has no tokens and needs to be refilled")
	}
	if err != nil {
		return false, common.NewError("invalid_request", fmt.Sprintf("getting faucet balance resulted in an error: %v", err.Error()))
	}
	if gn.PourAmount > smartContractBalance {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) exceeds contract's wallet ballance (%v)", t.ValueZCN, smartContractBalance))
	}
	if currency.Coin(gn.PourAmount)+un.Used > gn.PeriodicLimit {
		return false, common.NewError("invalid_request",
			fmt.Sprintf("amount asked to be poured (%v) plus previous amounts (%v) exceeds allowed periodic limit (%v/%vhr)",
				t.ValueZCN, un.Used, gn.PeriodicLimit, gn.IndividualReset.String()))
	}
	if currency.Coin(gn.PourAmount)+gn.Used > gn.GlobalLimit {
		return false, common.NewError("invalid_request",
			fmt.Sprintf("amount asked to be poured (%v) plus global used amount (%v) exceeds allowed global limit (%v/%vhr)",
				t.ValueZCN, gn.Used, gn.GlobalLimit, gn.GlobalReset.String()))
	}
	logging.Logger.Info("Valid sc request", zap.Any("contract_balance", smartContractBalance), zap.Any("txn.Value", t.ValueZCN), zap.Any("max_pour", gn.PourAmount), zap.Any("periodic_used+t.Value", currency.Coin(t.ValueZCN)+un.Used), zap.Any("periodic_limit", gn.PeriodicLimit), zap.Any("global_used+txn.Value", currency.Coin(t.ValueZCN)+gn.Used), zap.Any("global_limit", gn.GlobalLimit))
	return true, nil
}

func (fc *FaucetSmartContract) updateSettings(
	t *transaction.Transaction,
	inputData []byte,
	balances c_state.StateContextI,
	gn *GlobalNode,
) (string, error) {
	if err := smartcontractinterface.AuthorizeWithOwner("update_settings", func() bool {
		return gn.FaucetConfig.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var input sc.StringMap
	err := input.Decode(inputData)
	if err != nil {
		return "", common.NewError("update_settings", "limit request not formatted correctly")
	}

	if err := gn.updateConfig(input.Fields); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if err = gn.validate(); err != nil {
		return "", common.NewError("update_settings", "cannot validate changes: "+err.Error())
	}
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		return "", common.NewError("update_settings", "saving global node: "+err.Error())
	}
	return string(gn.Encode()), nil
}

func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

func (fc *FaucetSmartContract) pour(t *transaction.Transaction, _ []byte, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	user := fc.getUserVariables(t, gn, balances)
	ok, err := user.validPourRequest(t, balances, gn)
	if ok {
		var pourAmount = gn.PourAmount
		if t.ValueZCN > 0 && t.ValueZCN < int64(gn.MaxPourAmount) {
			pourAmount = currency.Coin(t.ValueZCN)
		}
		tokensPoured := fc.SmartContractExecutionStats["tokens Poured"].(metrics.Histogram)
		transfer := state.NewTransfer(t.ToClientID, t.ClientID, pourAmount)
		if err := balances.AddTransfer(transfer); err != nil {
			return "", err
		}
		user.Used += transfer.Amount
		gn.Used += transfer.Amount
		_, err = balances.InsertTrieNode(user.GetKey(gn.ID), user)
		if err != nil {
			return "", err
		}
		_, err := balances.InsertTrieNode(gn.GetKey(), gn)
		if err != nil {
			return "", err
		}
		tokensPoured.Update(int64(transfer.Amount))
		return string(transfer.Encode()), nil
	}
	return "", err
}

func (fc *FaucetSmartContract) refill(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	clientBalance, err := balances.GetClientBalance(t.ClientID)
	if err != nil {
		return "", err
	}
	if clientBalance >= currency.Coin(t.ValueZCN) {
		tokenRefills := fc.SmartContractExecutionStats["token refills"].(metrics.Histogram)
		transfer := state.NewTransfer(t.ClientID, t.ToClientID, currency.Coin(t.ValueZCN))
		if err := balances.AddTransfer(transfer); err != nil {
			return "", err
		}
		_, err := balances.InsertTrieNode(gn.GetKey(), gn)
		if err != nil {
			return "", err
		}
		tokenRefills.Update(int64(transfer.Amount))
		return string(transfer.Encode()), nil
	}
	return "", common.NewError("broke", "it seems you're broke and can't transfer money")
}

func (fc *FaucetSmartContract) getUserNode(id string, globalKey string, balances c_state.StateContextI) (*UserNode, error) {
	un := &UserNode{ID: id}
	err := balances.GetTrieNode(un.GetKey(globalKey), un)
	return un, err
}

func (fc *FaucetSmartContract) getUserVariables(t *transaction.Transaction, gn *GlobalNode, balances c_state.StateContextI) *UserNode {
	un, err := fc.getUserNode(t.ClientID, gn.ID, balances)
	if err != nil {
		un.StartTime = common.ToTime(t.CreationDate)
		un.Used = 0
	}
	if common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.IndividualReset ||
		common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.GlobalReset {
		un.StartTime = common.ToTime(t.CreationDate)
		un.Used = 0
	}
	return un
}

func (fc *FaucetSmartContract) getGlobalNode(balances c_state.StateContextI) (*GlobalNode, error) {
	gn := &GlobalNode{ID: fc.ID}
	err := balances.GetTrieNode(gn.GetKey(), gn)
	switch err {
	case nil, util.ErrValueNotPresent:
		if gn.FaucetConfig == nil {
			gn.FaucetConfig = getFaucetConfig()
		}
		return gn, err
	default:
		return nil, err
	}
}

func (fc *FaucetSmartContract) getGlobalVariables(t *transaction.Transaction, balances c_state.StateContextI) (*GlobalNode, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	if err == nil {
		if common.ToTime(t.CreationDate).Sub(gn.StartTime) >= gn.GlobalReset {
			gn.StartTime = common.ToTime(t.CreationDate)
			gn.Used = 0
		}
		return gn, nil
	}
	gn.Used = 0
	gn.StartTime = common.ToTime(t.CreationDate)
	return gn, nil
}

func (fc *FaucetSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn, err := fc.getGlobalVariables(t, balances)
	if err != nil {
		return "", common.NewError(funcName, "cannot get global node: "+err.Error())
	}

	switch funcName {
	case "update-settings":
		return fc.updateSettings(t, inputData, balances, gn)
	case "pour":
		return fc.pour(t, inputData, balances, gn)
	case "refill":
		return fc.refill(t, balances, gn)
	default:
		return "", common.NewErrorf("failed execution", "no faucet smart contract method with name %s", funcName)
	}
}
