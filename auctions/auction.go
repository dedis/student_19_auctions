package auctions

import (
	"encoding/binary"
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

		for i := 0; i < len(cin); i++ {
			//log.LLvl4(cin[i].Name)
			bid.Bid = bid.Bid + cin[i].Value
		}

		//log.LLvl4(bid.Bid)
		//todo
		// check coin name with rst getValues
		// In unit test, test with different coins name
		// Delete deposit account and just keep track of the money
		// At the end, do a store with a created coin array (c1 remember) with the same name and value
		// instead of transfer
		// Write test of test that can crash: bid with 0 can work? auction with no bid can it close? etc
		// !! List of the weeks left and my plans for finishing the project god

		if bid.Bid <= 0 { //can not bid 0 or less
			err = errors.New("can not bid 0 or less")
			return

		} else {
			if auction.HighestBid.Bid == 0 { //first bid
				//todo: incremental bid if same bidder
				sc, _, _ = c.storeCoin(rst, cin, auction.Deposit)
				if err != nil {
					err = errors.New("storeCoin")
					return
				}

				//proof, err := rst.GetProof(auction.Deposit.Slice())
				//if err != nil {
				//	return
				//}
				//
				//_, val := proof.KeyValue()

				//Then update highest bid/bidder
				auction.HighestBid = bid
				auctionBuf, err = protobuf.Encode(&auction)

				sc = append(sc, byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID))

			} else if bid.Bid > auction.HighestBid.Bid {

				//Refund old highest bidder
				bidCoins := make([]byte, 8)
				binary.LittleEndian.PutUint64(bidCoins, auction.HighestBid.Bid)

				sc, _, err = c.transferCoin(rst, bidCoins, auction.Deposit, auction.HighestBid.BidderAccount)
				if err != nil {
					return
				}

				//log.LLvl4("Transfer 1 work")

				//Store new highest bidder
				binary.LittleEndian.PutUint64(bidCoins, bid.Bid)
				//sc2, _, _ := c.storeCoin(rst, bidCoins, auction.Deposit)
				sc2, _, _ := c.storeCoin(rst, cin, auction.Deposit)
				//sc2, _, _ := c.storeCoin(rst, bid.Bid, auction.Deposit)
				if err != nil {
					err = errors.New("storeCoin")
					return
				}

				//log.LLvl4("Store 2 work")

				//Then update highest bid/bidder
				auction.HighestBid = bid
				auctionBuf, err = protobuf.Encode(&auction)

				sc = append(sc, byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
					ContractAuctionID, auctionBuf, darcID))

				sc = append(sc, sc2...)

			} else {
				err = errors.New("cannot bid less than current highest bid")
				return
			}
		}

	case "close":

		bidCoin := make([]byte, 8)
		//
		////Credit seller account
		binary.LittleEndian.PutUint64(bidCoin, auction.HighestBid.Bid)
		//.storeCoin()
		sc, cout, err = c.transferCoin(rst, bidCoin, auction.Deposit, auction.SellerAccount)
		if err != nil {
			return
		}

		auction.State = CLOSED

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

func (c *contractAuction) transferCoin(rst byzcoin.ReadOnlyStateTrie, amount []byte, debitAccount byzcoin.InstanceID, creditAccount byzcoin.InstanceID) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
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

	b := c.s.Service(byzcoin.ServiceName).(*byzcoin.Service)
	cFact, found := b.GetContractConstructor(contracts.ContractCoinID)
	if !found {
		err = errors.New("Not found")
		return
	}

	in, _, _, _, err := rst.GetValues(debitAccount.Slice())
	if err != nil {
		err = errors.New("cfactory getValues failed")
		return
	}

	cCoin, err := cFact(in)
	if err != nil {
		err = errors.New("coin factory failed")
		return
	}
	return cCoin.Invoke(rst, instruct, []byzcoin.Coin{})
}

func (c *contractAuction) storeCoin(rst byzcoin.ReadOnlyStateTrie, amount []byzcoin.Coin, creditAccount byzcoin.InstanceID) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
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
	return cCoin.Invoke(rst, instruct, amount)
}
