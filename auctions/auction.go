package auctions

import (
	"errors"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/protobuf"
)

// ContractAuctionID identifies an auction contract
var ContractAuctionID = "auction"

// ContractAuction
type contractAuction struct {
	byzcoin.BasicContract
	AuctionData
	s *byzcoin.Service
}

func contractAuctionFromBytes(in []byte) (byzcoin.Contract, error) {
	cv := &contractAuction{}
	err := protobuf.Decode(in, &cv.AuctionData)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

// Spawn creates a new auction (ContractAuction) instance
// It opens an auction and await for bids
// Spawn will store the arguments (good, seller account) in the data field.
func (c *contractAuction) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {

	cout = coins

	//Darc control the access to the connected instance
	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	//Verify contractAuctionID
	if inst.Spawn.ContractID != ContractAuctionID {
		return nil, nil, errors.New("can only spawn auction instances")
	}
	// Fill AuctionData structure
	// Put the data from the inst.Spawn.Args into our AuctionData structure.
	// auctionBuf store the value of the argument with name auction
	auctionBuf := inst.Spawn.Args.Search("auction")
	if auctionBuf == nil {
		return nil, nil, errors.New("need an argument with name auction")
	}

	//Verify that it's an auction
	auction := AuctionData{}
	err = protobuf.Decode(auctionBuf, &auction)
	if err != nil {
		return nil, nil, errors.New("Error: not an auction")
	}

	// Create the auction instance in the global state thanks to
	// a StateChange request with the data of the instance. The
	// InstanceID is given by the DeriveID method of the instruction that allows
	// to create multiple instanceIDs out of a given instruction in a pseudo-
	// random way that will be the same for all nodes.
	instID := inst.DeriveID("")
	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Create, instID, ContractAuctionID, auctionBuf, darcID),
	}
	return
}

// Override of function VerifyInstruction because
// The auction instance need to allow any user in the system to invoke “bid” on it. The default behaviour of the VerifyInstruction (see cothority/byzcoin/conrtacts.go line 58) is to try to find some signers in the instruction that satisfy the DARC that controls access to the instance. We need to override this behaviour to accept all bidders.
//func (c *contractAuction) VerifyInstruction(rst ReadOnlyStateTrie, inst Instruction, ctxHash []byte) error {
//	return nil
//}

// The following methods are available:
//  - bid: takes the bidders bid
//  - close: ends an auction
// You can only delete a contractAuction instance after the auction is closed.

func (c *contractAuction) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {

	cout = coins

	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	/*var ContractCoinID = "contracts"                                //"coins"
	accFactory, found := c.s.GetContractConstructor(ContractAuctionID) //-> I think only work with DARC Contracts
	if !found {
		return
	}*/

	if inst.Invoke.Command != "close" {
		bidBuf := inst.Invoke.Args.Search("bid")
		if bidBuf == nil {
			err = errors.New("Error: need an argument with name bid")
			return
		}
	}

	var auctionBuf []byte
	auctionBuf, _, _, _, err = rst.GetValues(inst.InstanceID.Slice())
	auction := AuctionData{}
	err = protobuf.Decode(auctionBuf, &auction)
	if err != nil {
		return
	}

	//Fill BidData structure
	//Put the data from the inst.Spawn.Args into our BidData structure.
	//bidBuf store the value of the argument with name bid
	bidBuf := inst.Invoke.Args.Search("bid")
	if bidBuf == nil {
		return nil, nil, errors.New("need an argument with name bid")
	}

	//Verify that it's a bid
	bid := BidData{}
	err = protobuf.Decode(bidBuf, &bid)
	if err != nil {
		return nil, nil, errors.New("Error: not a bid")
	}

	//fmt.Println(auction)
	//fmt.Println(bid)
	//var winner BidData

	if auction.State == "closed" {
		err = errors.New("Error: auction is closed")
		return
	}

	//// Invoke provides two methods "bid" or "close"
	switch inst.Invoke.Command {
	case "bid":
		if auction.State == "open" {

			auction.Bids = append(auction.Bids, bid)
			print(auction.Bids)

			var auctionBuf []byte
			auctionBuf, err = protobuf.Encode(&auction)
			if err != nil {
				return
			}

			sc = []byzcoin.StateChange{
				byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID),
			}
		}
	//case "close":
	//	log.Lvl2("closing")
	//	auctData.State = "closed"
	//	winner = getWinner(auctData.Bids)
	//	//Transferring the coins from winner to seller
	//	//trqnsfering coins is implemented in coins.go, but you need to
	//	//cqll into that by using GetContrqctConstructor to make a contract factory
	//	//then call it with the current value of the target account, which you get with GetValue.
	//	//example of the difficulty/solution of how to call GetContractConstructor in insecure_darc.go
	//
	default:
		err = errors.New("Auction contract can only bid or close")
	}

	return
}

//function getWinner
func getWinner(bids []BidData) BidData {
	var highestBid BidData = bids[0]
	for _, value := range bids {
		if highestBid.Bid < value.Bid {
			highestBid = value
		}
	}
	return highestBid
}
