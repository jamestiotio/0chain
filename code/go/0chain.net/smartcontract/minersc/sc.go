package minersc

import (
	"context"

	"github.com/asaskevich/govalidator"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/smartcontractstate"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
	"go.uber.org/zap"
	. "0chain.net/core/logging"
	"errors"
	"net/url"
)

const (
	//ADDRESS address of minersc
	ADDRESS = "CF9C03CD22C9C7B116EED04E4A909F95ABEC17E98FE631D6AC94D5D8420C5B20"
	bufRounds = 5000 //ToDo: make it configurable
	cfdBuffer = 10
)

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*smartcontractinterface.SmartContract
	bcContext smartcontractinterface.BCContextI
}

//SetSC setting up smartcontract. implementing the interface
func (msc *MinerSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	msc.SmartContract = sc
	msc.SmartContract.RestHandlers["/getNodepool"] = msc.GetNodepoolHandler
	msc.bcContext = bcContext
	
}

//Execute implemetning the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {

	switch funcName {

		case "add_miner": 
			resp, err := msc.AddMiner(t, input)
			if err != nil {
				return "", err
			}
			return resp, nil
		
		case "viewchange_req":
			resp, err := msc.RequestViewchange(t, input, balances)
			if err != nil {
				return "", err
			}
			return resp, nil
		
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	
	}
}

//REST API Handlers

//GetNodepoolHandler API to provide nodepool information for registered miners
func (msc *MinerSmartContract) GetNodepoolHandler(ctx context.Context, params url.Values) (interface{}, error){
	
	var regMiner MinerNode
	err := regMiner.decodeFromValues(params)
	if err != nil {
		Logger.Info("Returing error from GetNodePoolHandler", zap.Error(err))
		return nil, err	
	}
	if !msc.doesMinerExist(regMiner.getKey()) {
		return "", errors.New("unknown_miner"+err.Error())
	}
	npi := msc.bcContext.GetNodepoolInfo()
	
	return npi, nil
}

func (msc *MinerSmartContract) doesMinerExist(pkey smartcontractstate.Key) bool {
	mbits, _ := msc.DB.GetNode(pkey)
	if mbits != nil {
		return true
	}
	return false
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction, input []byte) (string, error) {
	
	allMinersList, err := msc.getMinersList()
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("add_miner_failed - Failed to get miner list"+err.Error())
	}

	var newMiner MinerNode
	err = newMiner.decode(input) 
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))
		
		return "", err
	}
	Logger.Info("The new miner info", zap.String("base URL", newMiner.BaseURL))
	newMiner.ID = t.ClientID
	newMiner.PublicKey = t.PublicKey
	
	if msc.doesMinerExist(newMiner.getKey()) {
		Logger.Info("Miner received already exist", zap.String("url", newMiner.BaseURL))
		
	} else {
		minerBytes, _ := msc.DB.GetNode(newMiner.getKey())
		if minerBytes == nil {
			//DB does not have the miner already. Validate before adding.
			err = isValidURL(newMiner.BaseURL) 
			
			if err != nil {
				Logger.Error (newMiner.BaseURL + "is not a valid URL. Please provide DNS name or IPV4 address")
				return "", errors.New(newMiner.BaseURL + "is not a valid URL. Please provide DNS name or IPV4 address")
			}
			//ToDo: Add clientID and publicKey validation
			allMinersList = append(allMinersList, newMiner)
			allMinersBytes, _ := json.Marshal(allMinersList)
			msc.DB.PutNode(allMinersKey, allMinersBytes)
			msc.DB.PutNode(newMiner.getKey(), newMiner.encode())
			Logger.Info("Adding miner to known list of miners", zap.Any("url", allMinersList))
		}  else {
			Logger.Info("Miner received already exist", zap.String("url", newMiner.BaseURL))
		}
	}
	buff := newMiner.encode()
	return string(buff), nil
}

//RequestViewchange Function to handle miner viewchange request
func (msc *MinerSmartContract) RequestViewchange(t *transaction.Transaction, input []byte, statectx c_state.StateContextI) (string, error) {
	
	var regMiner MinerNode
	err := regMiner.decode(input) 
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))
		
		return "", err
	}
	Logger.Info("The new view change request from", zap.String("base URL", regMiner.BaseURL))
	regMiner.ID = t.ClientID
	regMiner.PublicKey = t.PublicKey
	
	if !msc.doesMinerExist(regMiner.getKey()) {
		Logger.Info("Miner received does not exist", zap.String("url", regMiner.BaseURL))
		return "", errors.New( regMiner.BaseURL + " Miner rdoes not exist",)
	} 


	curRound := statectx.GetBlock().Round
	vcRound := (((int64)((curRound+bufRounds)/1000)) + 1) * 1000
	vcRoundInfo := &ViewchangeInfo{}

	vcRoundInfo.ViewchangeRound = vcRound
	vcRoundInfo.ViewchangeCFDRound = vcRound - cfdBuffer

	Logger.Info("RequestViewChange", zap.Int64("cur_round", curRound), 
					zap.Int64("vc_round", vcRoundInfo.ViewchangeRound), zap.Int64("dkg_round", vcRoundInfo.ViewchangeCFDRound))

	buff := vcRoundInfo.encode()
	return string(buff), nil
	
	
}





//------------- local functions ---------------------

func (msc *MinerSmartContract) getMinersList() ([]MinerNode, error) {
	var allMinersList = make([]MinerNode, 0)
	allMinersBytes, err := msc.DB.GetNode(allMinersKey)
	if err != nil {
		return nil, errors.New("getMinersList_failed - Failed to retrieve existing miners list")
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	err = json.Unmarshal(allMinersBytes, &allMinersList)
	if err != nil {
		return nil, errors.New("getBlobbersList_failed - Failed to retrieve existing blobbers list")
	}
	return allMinersList, nil
}

func isValidURL(burl string) error {
	//ToDo: does rudimentary checks. Add more checks
	u, err := url.Parse(burl)
	if err != nil {
		return errors.New(burl + " is not a valid url")
	}

	if u.Scheme != "http" { //|| u.scheme == "https"  we don't support
		return errors.New(burl + " is not a valid url. It does not have scheme http")
	}

	if u.Port() == "" {
		return errors.New(burl + " is not a valid url. It does not have port number")
	}

	h := u.Hostname()
	
	if govalidator.IsDNSName(h)  {
		return nil
	} 
	if govalidator.IsIPv4(h) {
		return nil
	}
	Logger.Info("Both IsDNSName and IsIPV4 returned false for " + h)
	return errors.New(burl + " is not a valid url. It not a valid IP or valid DNS name")

}