package sb_auctions

import (
	"encoding/binary"
	"errors"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/protobuf"
)

// ContractAuctionID identifies an auction contract
var ContractSBAuctionID = "sb_auction"

type contractSBAuction struct {
	byzcoin.BasicContract
	AuctionData
	s *byzcoin.Service
}

func contractSBAuctionFromBytes(in []byte) (byzcoin.Contract, error) {
	cv := &contractSBAuction{}
	err := protobuf.Decode(in, &cv.AuctionData)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

// Spawn creates a new auction (ContractAuction) instance
// It opens an auction and await for bids
// Spawn will store the arguments (good, seller account) in the data field.
func (c *contractSBAuction) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {

	cout = coins

	//Darc control the access to the connected instance
	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, err
	}

	//Verify contractAuctionID
	if inst.Spawn.ContractID != ContractSBAuctionID {
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
		byzcoin.NewStateChange(byzcoin.Create, auctInstID, ContractSBAuctionID, auctionBuf, darcID),
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

func (c *contractSBAuction) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {

	cout = coins

	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	if inst.Invoke.Command == "bid" {
		bidBuf := inst.Invoke.Args.Search("bid")
		if bidBuf == nil {
			err = errors.New("need an argument with name bid")
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

	if auction.State == CLOSED && inst.Invoke.Command == "bid" {
		err = errors.New("auction is closed, cannot bid")
		return nil, nil, err
	}

	//// Invoke provides two methods "bid", "close" or "process"
	switch inst.Invoke.Command {
	case "bid":
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
			return nil, nil, errors.New("not a bid")
		}

		found, i := c.searchBidder(auction.Bids, bid.BidderAccount)
		if found != true { //first bid
			bidCoins := make([]byte, 8)
			binary.BigEndian.PutUint32(bidCoins, bid.Bid)
			if err != nil {
				err = errors.New("making converting bid to coins")
				return nil, nil, err
			}
			//err := c.transferCoin(rst, bidCoins, bid.BidderAccount, auction.Deposits)
			//if err != nil {
			//	err = errors.New("transfer coins not working")
			//	return nil, nil, err
			//}
			auction.Bids = append(auction.Bids, bid)

		} else { //update bid
			prevBid := auction.Bids[i].Bid
			if bid.Bid < prevBid {
				err = errors.New("cannot bid less than previous bid")
				return nil, nil, err
			} else {
				//Incremental bid
				diff := bid.Bid - prevBid
				bidCoins := make([]byte, 8)
				binary.BigEndian.PutUint32(bidCoins, diff)

				//err := c.transferCoin(rst, bidCoins, bid.BidderAccount, auction.Deposits)
				//if err != nil {
				//	err = errors.New("not enough coins to update bid")
				//	return nil, nil, err
				//}
				auction.Bids[i].Bid = bid.Bid

			}
		}

		auctionBuf, err = protobuf.Encode(&auction)
		if err != nil {
			return nil, nil, errors.New("encode auction buf sc")
		}

		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				ContractSBAuctionID, auctionBuf, darcID),
		}

	case "close":
		auction.State = CLOSED

		auctionBuf, err = protobuf.Encode(&auction)
		if err != nil {
			return nil, nil, errors.New("encode auction buf sc")
		}

		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				ContractSBAuctionID, auctionBuf, darcID),
		}

	case "process":
		var winner BidData
		winner, auction.Bids = getWinner(auction.Bids)

		if winner.Bid <= auction.ReservePrice {
			err = errors.New("Reserve price not reached")
			return nil, nil, err
		}

		//auction.Bids, err = c.creditAndRefundCoin(rst, winner, auction.SellerAccount, auction.Bids, auction.Deposits)
		//if err != nil {
		//	err = errors.New("refunding not working")
		//	return nil, nil, err
		//}

		auction.WinnerAccount = winner.BidderAccount

		auctionBuf, err = protobuf.Encode(&auction)
		if err != nil {
			return nil, nil, errors.New("encode auction buf sc")
		}

		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				ContractSBAuctionID, auctionBuf, darcID),
		}

	default:
		err = errors.New("Auction contract can only bid, close or process")
	}

	return
}

//function getWinner
func getWinner(bids []BidData) (BidData, []BidData) {
	var highestBid BidData = bids[0]
	index := 0
	for i, value := range bids {
		if highestBid.Bid < value.Bid {
			highestBid = value
			index = i
		}
	}
	bids = remove(bids, index)
	return highestBid, bids
}

func (c *contractSBAuction) transferCoin(rst byzcoin.ReadOnlyStateTrie, amount []byte, debitAccount byzcoin.InstanceID, creditAccount byzcoin.InstanceID) (err error) {
	instruct := byzcoin.Instruction{
		InstanceID: debitAccount,
		Invoke: &byzcoin.Invoke{
			ContractID: contracts.ContractCoinID,
			Command:    "transfer",
			Args: byzcoin.Arguments{
				{Name: "coins", Value: amount},
				{Name: "destination", Value: creditAccount.Slice()},
			},
		},
	}

	cFact, found := c.s.GetContractConstructor(contracts.ContractCoinID)
	if !found {
		return err
	}

	cCoin, err := cFact(nil)
	if err != nil {
		return err
	}
	_, _, err = cCoin.Invoke(rst, instruct, []byzcoin.Coin{})
	if err != nil {
		return err
	}

	return
	//log.LLvl4(s)
}

func (c *contractSBAuction) creditAndRefundCoin(rst byzcoin.ReadOnlyStateTrie, winner BidData, sellerAccount byzcoin.InstanceID, bids []BidData, depositAccount byzcoin.InstanceID) (b []BidData, err error) {

	bidCoins := make([]byte, 8)

	//Credit seller account
	binary.BigEndian.PutUint32(bidCoins, winner.Bid)
	err = c.transferCoin(rst, bidCoins, depositAccount, sellerAccount)
	if err != nil {
		err = errors.New("transfer coins deposit to seller not working")
		return bids, err
	}

	//Refund
	for i, bid := range bids {
		binary.BigEndian.PutUint32(bidCoins, bid.Bid)
		err := c.transferCoin(rst, bidCoins, depositAccount, bid.BidderAccount)
		if err != nil {
			return bids, err
		}
		bids = remove(bids, i)
	}

	return bids, err
}

func remove(bids []BidData, i int) []BidData {
	length := len(bids) - 1
	bids[i] = bids[length]   // Copy last element to index i.
	bids[length] = BidData{} // Erase last element
	bids = bids[:length]
	return bids
}

func (c *contractSBAuction) searchBidder(bids []BidData, bidAcc byzcoin.InstanceID) (bool, int) {
	for i, bid := range bids {
		if bid.BidderAccount == bidAcc {
			return true, i
		}
	}
	return false, 0
}
