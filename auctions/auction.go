package auctions

import (
	"errors"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
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

	if auction.State == CLOSED {
		err = errors.New("auction is closed, cannot bid")
		return nil, nil, err
	}

	//// Invoke provides two methods "bid" or "close"
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

		if bid.Bid > auction.HighestBid.Bid {

			//bidCoins := make([]byte, 8)
			//
			////First check if new bidder has enough coins
			//binary.BigEndian.PutUint32(bidCoins, bid.Bid)
			//err := c.transferCoin(rst, bidCoins, bid.BidderAccount, auction.Deposit)
			//if err != nil {
			//	err = errors.New("not enough coins to bid")
			//	return nil, nil, err
			//}
			//
			////Second, refund old highest bidder
			//binary.BigEndian.PutUint32(bidCoins, auction.HighestBid.Bid)
			//err = c.transferCoin(rst, bidCoins, auction.Deposit, auction.HighestBid.BidderAccount)
			//if err != nil {
			//	err = errors.New("refunding old highest bidder not working")
			//	return nil, nil, err
			//}

			//Then update highest bid/bidder
			auction.HighestBid = bid

			auctionBuf, err = protobuf.Encode(&auction)
			if err != nil {
				return nil, nil, errors.New("encode auction buf sc")
			}

			sc = []byzcoin.StateChange{
				byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID),
			}
		} else {
			err = errors.New("cannot bid less than current highest bid")
			return nil, nil, err
		}

	case "close":

		//bidCoins := make([]byte, 8)
		//
		////Credit seller account
		//binary.BigEndian.PutUint32(bidCoins, auction.HighestBid.Bid)
		//err := c.transferCoin(rst, bidCoins, auction.Deposit, auction.SellerAccount)
		//if err != nil {
		//	err = errors.New("not enough coins to bid")
		//	return nil, nil, err
		//}

		auction.State = CLOSED

		auctionBuf, err = protobuf.Encode(&auction)
		if err != nil {
			return nil, nil, errors.New("encode auction buf sc: close")
		}

		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				ContractAuctionID, auctionBuf, darcID),
		}

	default:
		err = errors.New("Auction contract can only bid or close")
	}

	return
}

func (c *contractAuction) transferCoin(rst byzcoin.ReadOnlyStateTrie, amount []byte, debitAccount byzcoin.InstanceID, creditAccount byzcoin.InstanceID) (err error) {
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
