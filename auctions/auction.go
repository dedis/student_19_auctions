package auctions

import (
	"encoding/binary"
	"errors"

	"go.dedis.ch/onet/v3/log"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/protobuf"
)

// ContractAuctionID identifies an auction contract
var ContractAuctionID = "auction"

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
		return nil, nil, err
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
	auctInstID := inst.DeriveID("")
	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Create, auctInstID, ContractAuctionID, auctionBuf, darcID),
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

	if auction.State == CLOSED {
		err = errors.New("Error: auction is closed")
		return
	}

	//// Invoke provides two methods "bid" or "close"
	switch inst.Invoke.Command {
	case "bid":
		if auction.State == OPEN {

			bidCoins := make([]byte, 8)
			binary.BigEndian.PutUint32(bidCoins, bid.Bid)
			if err != nil {
				err = errors.New("making converting bid to coins")
				return
			}

			auction.Bids = append(auction.Bids, bid)

			var auctionBuf []byte
			auctionBuf, err = protobuf.Encode(&auction)
			if err != nil {
				log.LLvl4(err)
				return
			}

			sc = []byzcoin.StateChange{
				byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID),
			}

			//_, _, err := getContract(instruct.InstanceID).Invoke(rst, instruct, []byzcoin.Coin{})
			//if err != nil {
			//	fmt.Println("Error invoke transfert, bidder can't bid")
			//	fmt.Println(err)
			//}

			//accFactory, found := c.s.GetContractConstructor(contracts.ContractCoinID)
			//if !found {
			//	fmt.Println("Error invoke transfert: factory")
			//}
			//var acc byzcoin.Contract
			//acc, err = accFactory(nil)
			//if err != nil {
			//	return nil, nil, fmt.Errorf("coult not spawn new zero instance: %v", err)
			//}
			//// _, _, err = acc.Invoke(rst, instruct, []byzcoin.Coin{})
			//
			//auction.Bids = append(auction.Bids, bid)
			//log.LLvl4("the bids are", auction.Bids)
			//

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
