package byzcoin

import (
	"errors"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
)

// ContractAuctionID identifies an auction contract
var ContractAuctionID = "auction"

// ContractAuction
type contractAuction struct {
	byzcoin.BasicContract
	AuctionData
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
			return nil, nil, errors.New("Error: can only spawn auction instances")
	}
	// Fill AuctionData structure
	// Put the data from the inst.Spawn.Args into our AuctionData structure.
	// auctionBuf store the value of the argument with name auction
	auctionBuf := inst.Spawn.Args.Search("auction")
	if auctionBuf == nil {
		return nil, nil, errors.New("Error: need an argument with name auction")
	}

	//Verify that it's an auction
	auction := AuctionData{}
	err = protobuf.Decode(auctionBuf, &auction)
	if err != nil {
		return nil, nil, errors.New("Error: not an auction")
	}

	auctData := &c.AuctionData
	auctData.GoodDescription = auction.GoodDescription
	auctData.SellerAccount = auction.SellerAccount
	auctData.Bids = auction.Bids
	auctData.State = auction.State


	auctInstBuf, err := protobuf.Encode(&c.AuctionData)
	if err != nil {
		return nil, nil, errors.New("Error: couldn't encode AuctionInstance: " + err.Error())
	}

	// Create the auction instance in the global state thanks to
	// a StateChange request with the data of the instance. The
	// InstanceID is given by the DeriveID method of the instruction that allows
	// to create multiple instanceIDs out of a given instruction in a pseudo-
	// random way that will be the same for all nodes.
	instID := inst.DeriveID("")
	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Create, instID, ContractAuctionID, auctInstBuf, darcID),
	}
	return
}

// Override of function VerifyInstruction because
// The auction instance need to allow any user in the system to invoke “bid” on it. The default behaviour of the VerifyInstruction (see cothority/byzcoin/conrtacts.go line 58) is to try to find some signers in the instruction that satisfy the DARC that controls access to the instance. We need to override this behaviour to accept all bidders.
func (c *contractAuction) VerifyInstruction(rst ReadOnlyStateTrie, inst Instruction, ctxHash []byte) error {
	return nil
}

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

	auctData := &c.AuctionData
	var winner BidData

	if auctData.State == "closed" {
		err = errors.New("Error: auction is closed")
		return
	}
	// Invoke provides two methods "bid" or "close"
	switch inst.Invoke.Command {
		case "bid":
			if auctData.State == "open" {
				log.Lvl2("New bid")
				// Check if bidders has enough coins
				test = bidBuf.BidderAccount.SafeSub(bidBuf.Bid)
				if test != nil {
					log.Lvl2("Bid not accepted. Not enough coins")
					return
				}
				//Problem
				auctData.Bids = append(auctData.Bids, bidBuf)
				return
			}		else{
				err = errors.New("Error: auction is not open")
			}
		case "close":
			log.Lvl2("closing")
			auctData.Status = "closed"
			winner = getWinner(auctData.Bids)
			//Transferring the coins from winner to seller
			test = auctData.SellerAccount.SafeAdd(winner.Deposit)
			if test != nil {
				return
			}
			sc = append(sc,...)

		default:
			err = errors.New("Auction contract can only bid or close")
			return
	}

	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
			ContractKeyValueID, buf, darcID),
	}
	return
}

//function getWinner
func getWinner(bids []BidData) (BidData) {
	var highestBid BidData = bids[0]
	for _, value := range bids {
		if highestBid.Bid < value.Bid {
			highestBid = value
		}
	}
	return highestBid
}

//Delete the auction instance
func (c *contractAuction) Delete(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {

	cout = coins

	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	auctData := &c.AuctionData
	if auctData.Status != "closed" {
		err = errors.New("Error: cannot destroy an auctionInstance that is still active")
		return
	}

	// Delete removes all the data from the global state.
	sc = byzcoin.StateChanges{byzcoin.NewStateChange(byzcoin.Remove, inst.InstanceID, ContractAuctionID, nil, darcID)}
	return
}
