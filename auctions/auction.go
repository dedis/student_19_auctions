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

type ContractCoin struct {
	byzcoin.BasicContract
	byzcoin.Coin
}

type contractAuction struct {
	byzcoin.BasicContract
	AuctionData
	s *Service
}

func (s *Service) contractAuctionFromBytes(in []byte) (byzcoin.Contract, error) {
	cv := &contractAuction{}
	err := protobuf.Decode(in, &cv.AuctionData)
	if err != nil {
		return nil, err
	}
	cv.s = s
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
		//return nil, nil, errors.New("Error: not an auction")
		return
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

func (c *contractAuction) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, cin []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
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

	if auction.State == "CLOSED" {
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
			err = errors.New("need an argument with name bid")
			return
		}

		//Verify that it's a bid
		bid := BidData{}
		err = protobuf.Decode(bidBuf, &bid)
		if err != nil {
			err = errors.New("not a bid")
			return
		}

		//If seller bids -> forbidden
		if bid.BidderAccount == auction.SellerAccount {
			err = errors.New("seller can not bid")
			return
		}

		//Check the coin name
		val, _, _, _, _ := rst.GetValues(bid.BidderAccount.Slice())
		coinS := ContractCoin{} //need this struct
		//var coinS struct{}
		err = protobuf.Decode(val, &coinS)
		if err != nil {
			return
		}
		//log.LLvl4("Coin name:", coinS.Name)

		for i := 0; i < len(cin); i++ {
			//log.LLvl4(cin[i].Name)
			if cin[i].Name == coinS.Name {
				bid.Bid = bid.Bid + cin[i].Value
			}
		}

		//log.LLvl4(bid.Bid)

		if bid.Bid <= 0 { //can not bid 0 or less
			err = errors.New("can not bid 0 or less")
			return

		} else {
			if auction.HighestBid == 0 { //first bid

				auction.HighestBid = bid.Bid
				auction.HighestBidder = bid.BidderAccount
				auctionBuf, err = protobuf.Encode(&auction)

				sc = append(sc, byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID))

			} else if bid.Bid > auction.HighestBid {
				//Refund old highest bidder
				sc, _, err = c.storeCoin(rst, auction.HighestBid, auction.HighestBidder)
				if err != nil {
					return
				}

				//Then update highest bid/bidder
				auction.HighestBid = bid.Bid
				auction.HighestBidder = bid.BidderAccount
				auctionBuf, err = protobuf.Encode(&auction)

				sc = append(sc, byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID))

			} else {
				err = errors.New("cannot bid less than current highest bid")
				return
			}
		}

	case "close":

		if auction.HighestBid > 0 {
			sc, cout, err = c.storeCoin(rst, auction.HighestBid, auction.SellerAccount)
			if err != nil {
				return
			}
		}
		auction.State = "CLOSED"

		auctionBuf, err = protobuf.Encode(&auction)
		if err != nil {
			return nil, nil, errors.New("encode auction buf sc: close")
		}

		sc = append(sc, byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
			ContractAuctionID, auctionBuf, darcID))

	default:
		err = errors.New("Auction contract can only bid or close")
	}

	return
}

func (c *contractAuction) storeCoin(rst byzcoin.ReadOnlyStateTrie, amount uint64, creditAccount byzcoin.InstanceID) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
	instruct := byzcoin.Instruction{
		InstanceID: creditAccount,
		Invoke: &byzcoin.Invoke{
			ContractID: contracts.ContractCoinID,
			Command:    "store",
			Args:       nil,
		},
	}

	b := c.s.Service(byzcoin.ServiceName).(*byzcoin.Service)
	cFact, found := b.GetContractConstructor(contracts.ContractCoinID)
	if !found {
		err = errors.New("Not found")
		return
	}

	in, _, _, _, err := rst.GetValues(creditAccount.Slice())
	if err != nil {
		err = errors.New("cfactory getValues failed")
		return
	}

	cCoin, err := cFact(in)
	if err != nil {
		err = errors.New("coin factory failed")
		return
	}

	coinS := ContractCoin{} //need this struct
	err = protobuf.Decode(in, &coinS)
	if err != nil {
		return
	}

	c1 := byzcoin.Coin{Name: coinS.Name, Value: amount}
	//log.LLvl4("c1 name", c1.Name)

	return cCoin.Invoke(rst, instruct, []byzcoin.Coin{c1})
}
