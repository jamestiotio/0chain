package storagesc

import (
	"encoding/json"
	"fmt"

	state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
)

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances state.StateContextI) (string, error) {
	newValidator := &ValidationNode{}
	err := newValidator.Decode(input) //json.Unmarshal(input, &newValidator)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey

	tmp := &ValidationNode{}
	err = balances.GetTrieNode(newValidator.GetKey(sc.ID), tmp)
	switch err {
	case nil:
		sc.statIncr(statUpdateValidator)
	case util.ErrValueNotPresent:
		_, err = sc.getBlobber(newValidator.ID, balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"new validator id does not match a registered blobber: "+err.Error())
		}

		validatorPartitions, err := getValidatorsList(balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator list."+err.Error())
		}

		_, err = validatorPartitions.AddItem(
			balances,
			&ValidationPartitionNode{
				Id:  t.ClientID,
				Url: newValidator.BaseURL,
			})
		if err != nil {
			return "", err
		}

		if err := validatorPartitions.Save(balances); err != nil {
			return "", err
		}

		_, err = balances.InsertTrieNode(newValidator.GetKey(sc.ID), newValidator)
		if err != nil {
			return "", err
		}

		sc.statIncr(statAddValidator)
		sc.statIncr(statNumberOfValidators)
	default:
		return "", common.NewError("add_validator_failed",
			"Failed to get validator."+err.Error())
	}

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrUpdateStakePool(conf, t.ClientID, spenum.Validator,
		newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", common.NewError("add_validator_failed",
			"get or create stake pool error: "+err.Error())
	}
	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("add_validator_failed",
			"saving stake pool error: "+err.Error())
	}
	data, _ := json.Marshal(dbs.DbUpdates{
		Id: t.ClientID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, t.ClientID, string(data))

	err = emitAddOrOverwriteValidatorTable(newValidator, balances, t)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emmiting Validation node failed: %v", err.Error())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}

func getValidator(
	validatorID string,
	balances state.CommonStateContextI,
) (*ValidationNode, error) {
	validator := new(ValidationNode)
	validator.ID = validatorID
	err := balances.GetTrieNode(validator.GetKey(ADDRESS), validator)
	if err != nil {
		return nil, err
	}
	return validator, nil
}

func (_ *StorageSmartContract) getValidator(
	validatorID string,
	balances state.StateContextI,
) (validator *ValidationNode, err error) {
	return getValidator(validatorID, balances)
}

func (sc *StorageSmartContract) updateValidatorSettings(t *transaction.Transaction, input []byte, balances state.StateContextI) (string, error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get config: "+err.Error())
	}

	var updatedValidator = new(ValidationNode)
	if err = updatedValidator.Decode(input); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"malformed request: "+err.Error())
	}

	var validator *ValidationNode
	if validator, err = sc.getValidator(updatedValidator.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get the validator: "+err.Error())
	}

	var sp *stakePool
	if sp, err = sc.getStakePool(updatedValidator.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get related stake pool: "+err.Error())
	}

	if sp.Settings.DelegateWallet == "" {
		return "", common.NewError("update_validator_settings_failed",
			"validator's delegate_wallet is not set")
	}

	if t.ClientID != sp.Settings.DelegateWallet {
		return "", common.NewError("update_validator_settings_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	if err = sc.updateValidator(t, conf, updatedValidator, validator, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed", err.Error())
	}
	validator.StakePoolSettings = updatedValidator.StakePoolSettings

	// save validator
	_, err = balances.InsertTrieNode(validator.GetKey(sc.ID), validator)
	if err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"saving validator: "+err.Error())
	}

	if err := emitUpdateValidator(validator, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"emitting update validator: "+err.Error())
	}

	return string(validator.Encode()), nil
}

func (sc *StorageSmartContract) hasValidatorUrl(validatorURL string,
	balances state.StateContextI) bool {
	validator := new(ValidationNode)
	validator.BaseURL = validatorURL
	err := balances.GetTrieNode(validator.GetUrlKey(sc.ID), &datastore.NOIDField{})
	return err == nil
}

// update existing validator, or reborn a deleted one
func (sc *StorageSmartContract) updateValidator(t *transaction.Transaction,
	conf *Config, validator *ValidationNode, savedValidator *ValidationNode,
	balances state.StateContextI,
) (err error) {
	// check params
	if err = validator.validate(conf); err != nil {
		return fmt.Errorf("invalid validator params: %v", err)
	}

	if savedValidator.BaseURL != validator.BaseURL {
		//if updating url
		if sc.hasValidatorUrl(validator.BaseURL, balances) {
			return fmt.Errorf("invalid validator url update, already used")
		}
		// save url
		if validator.BaseURL != "" {
			_, err = balances.InsertTrieNode(validator.GetUrlKey(sc.ID), &datastore.NOIDField{})
			if err != nil {
				return fmt.Errorf("saving validator url: " + err.Error())
			}
		}
		// remove old url
		if savedValidator.BaseURL != "" {
			_, err = balances.DeleteTrieNode(savedValidator.GetUrlKey(sc.ID))
			if err != nil {
				return fmt.Errorf("deleting validator old url: " + err.Error())
			}
		}
	}

	// update statistics
	sc.statIncr(statUpdateValidator)

	// update stake pool settings
	var sp *stakePool
	if sp, err = sc.getStakePool(validator.ID, balances); err != nil {
		return fmt.Errorf("can't get stake pool:  %v", err)
	}

	if err = validateStakePoolSettings(validator.StakePoolSettings, conf); err != nil {
		return fmt.Errorf("invalid new stake pool settings:  %v", err)
	}

	sp.Settings.MinStake = validator.StakePoolSettings.MinStake
	sp.Settings.MaxStake = validator.StakePoolSettings.MaxStake
	sp.Settings.ServiceChargeRatio = validator.StakePoolSettings.ServiceChargeRatio
	sp.Settings.MaxNumDelegates = validator.StakePoolSettings.MaxNumDelegates

	if err := emitAddOrOverwriteValidator(validator, sp, balances); err != nil {
		return fmt.Errorf("emmiting validator %v: %v", validator, err)
	}

	// save stake pool
	if err = sp.save(sc.ID, validator.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	data, _ := json.Marshal(dbs.DbUpdates{
		Id: validator.ID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateValidator, validator.ID, string(data))

	return
}
