package auctions

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/protobuf"
)

// bcTest is used here to provide some simple test structure for different
// tests.
type bcTest struct {
	local   *onet.LocalTest
	signer  darc.Signer
	servers []*onet.Server
	roster  *onet.Roster
	cl      *byzcoin.Client
	gMsg    *byzcoin.CreateGenesisBlock
	gDarc   *darc.Darc
	ct      uint64
}

func newBCTest(t *testing.T) (out *bcTest) {
	out = &bcTest{}
	// First create a local test environment with three nodes.
	out.local = onet.NewTCPTest(cothority.Suite)

	out.signer = darc.NewSignerEd25519(nil, nil)
	out.servers, out.roster, _ = out.local.GenTree(3, true)

	// Then create a new ledger with the genesis darc having the right
	// to create and update keyValue contracts.
	var err error
	out.gMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, out.roster,
		[]string{"spawn:auction", "invoke:auction.bid", "spawn:coin", "invoke:coin.mint"}, out.signer.Identity())
	require.Nil(t, err)
	out.gDarc = &out.gMsg.GenesisDarc

	// This BlockInterval is good for testing, but in real world applications this
	// should be more like 5 seconds.
	out.gMsg.BlockInterval = time.Second / 2

	out.cl, _, err = byzcoin.NewLedger(out.gMsg, false)
	require.Nil(t, err)
	out.ct = 1

	return out
}

func (bct *bcTest) Close() {
	bct.local.CloseAll()
}

func (bct *bcTest) createInstance(t *testing.T, args byzcoin.Arguments) byzcoin.InstanceID {
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID:    byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
			SignerCounter: []uint64{bct.ct},
			Spawn: &byzcoin.Spawn{
				ContractID: ContractAuctionID,
				Args:       args,
			},
		}},
	}
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))
	bct.ct++

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	return ctx.Instructions[0].DeriveID("")
}

func (bct *bcTest) updateInstance(t *testing.T, instID byzcoin.InstanceID, args byzcoin.Arguments) {
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID:    instID,
			SignerCounter: []uint64{bct.ct},
			Invoke: &byzcoin.Invoke{
				ContractID: ContractAuctionID,
				Command:    "update",
				Args:       args,
			},
		}},
	}
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))
	bct.ct++
	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
}

func (bct *bcTest) createSellerAndDepositAccount(t *testing.T) (byzcoin.InstanceID, byzcoin.InstanceID) {
	inst := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.ContractCoinID,
		},
		SignerIdentities: []darc.Identity{bct.signer.Identity()},
		SignerCounter:    []uint64{bct.ct},
	}

	inst1 := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.ContractCoinID,
		},
		SignerIdentities: []darc.Identity{bct.signer.Identity()},
		SignerCounter:    []uint64{bct.ct + 1},
	}

	ctx := byzcoin.ClientTransaction{Instructions: byzcoin.Instructions{inst, inst1}}
	err := ctx.FillSignersAndSignWith(bct.signer)
	require.NoError(t, err)

	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)
	bct.ct += 2

	sellAccInstID := ctx.Instructions[0].DeriveID("")
	depAccInstID := ctx.Instructions[1].DeriveID("")

	return sellAccInstID, depAccInstID
}

func (bct *bcTest) createBidderAccount(t *testing.T, amount uint32) byzcoin.InstanceID {
	inst := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.ContractCoinID,
		},
		SignerIdentities: []darc.Identity{bct.signer.Identity()},
		SignerCounter:    []uint64{bct.ct},
	}
	bct.ct += 1

	ctx := byzcoin.ClientTransaction{Instructions: byzcoin.Instructions{inst}}
	err := ctx.FillSignersAndSignWith(bct.signer)
	require.NoError(t, err)

	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	bidAccInstID := ctx.Instructions[0].DeriveID("")

	credit := make([]byte, 8)
	binary.BigEndian.PutUint32(credit, amount)

	inst = byzcoin.Instruction{
		InstanceID: bidAccInstID,
		Invoke: &byzcoin.Invoke{
			ContractID: contracts.ContractCoinID,
			Command:    "mint",
			Args:       byzcoin.Arguments{{Name: "coins", Value: credit}},
		},
		SignerIdentities: []darc.Identity{bct.signer.Identity()},
		SignerCounter:    []uint64{bct.ct},
	}

	bct.ct += 1

	ctx = byzcoin.ClientTransaction{Instructions: byzcoin.Instructions{inst}}
	err = ctx.FillSignersAndSignWith(bct.signer)
	require.NoError(t, err)

	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	return bidAccInstID
}

func (bct *bcTest) createAuction(t *testing.T, sellAccInstID byzcoin.InstanceID, depAccInstID byzcoin.InstanceID, good string, reservePrice uint32) (byzcoin.InstanceID, AuctionData) {
	auction := AuctionData{
		GoodDescription: good,
		SellerAccount:   sellAccInstID,
		ReservePrice:    reservePrice,
		Bids:            []BidData{},
		State:           OPEN,
		WinnerAccount:   byzcoin.InstanceID{},
		Deposits:        depAccInstID,
	}

	auctionBuf, err := protobuf.Encode(&auction)
	if err != nil {
		t.Fatal(err)
	}

	// Spawning new auction
	auctionArgs := byzcoin.Arguments{
		{
			Name:  "auction",
			Value: auctionBuf,
		},
	}
	auctInstID := bct.createInstance(t, auctionArgs)

	return auctInstID, auction
}

func (bct *bcTest) createBid(t *testing.T, auctInstID byzcoin.InstanceID, bidAccInstID byzcoin.InstanceID, bid uint32) BidData {
	bidata := BidData{
		BidderAccount: bidAccInstID,
		prevBid:       0,
		Bid:           bid,
	}

	bidBuf, err := protobuf.Encode(&bidata)
	if err != nil {
		t.Fatal(err)
	}

	// Try to invoke
	bidArgs := byzcoin.Arguments{
		{
			Name:  "bid",
			Value: bidBuf,
		},
	}

	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID: auctInstID,
			Invoke: &byzcoin.Invoke{
				ContractID: ContractAuctionID,
				Command:    "bid",
				Args:       bidArgs,
			},
			SignerCounter: []uint64{bct.ct},
		}},
	}

	bct.ct += 1

	require.Nil(t, ctx.FillSignersAndSignWith(bct.signer))
	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	return bidata
}

func (bct *bcTest) verifCreateAuction(t *testing.T, auctInstID byzcoin.InstanceID, auction AuctionData) AuctionData {

	auctS := bct.proofAndDecodeAuction(t, auctInstID)

	// Verify value.
	require.Equal(t, auction.GoodDescription, auctS.GoodDescription)
	require.Equal(t, auction.SellerAccount, auctS.SellerAccount)
	require.Equal(t, auction.State, auctS.State)

	return auctS

}

func (bct *bcTest) verifAddBidToAuction(t *testing.T, auctInstID byzcoin.InstanceID, auction AuctionData, bids []BidData) AuctionData {

	auctS := bct.proofAndDecodeAuction(t, auctInstID)

	// Verify value
	require.Equal(t, bids, auctS.Bids)

	return auctS

}

func (bct *bcTest) proofAndDecodeAuction(t *testing.T, auctInstID byzcoin.InstanceID) AuctionData {
	//Get the proof from byzcoin
	reply, err := bct.cl.GetProof(auctInstID.Slice())
	require.Nil(t, err)
	// Make sure the proof is a matching proof and not a proof of absence.
	proof := reply.Proof
	require.True(t, proof.InclusionProof.Match(auctInstID.Slice()))

	// Get the raw values of the proof.
	_, val, _, _, err := proof.KeyValue()
	require.Nil(t, err)

	// And decode the buffer to a AuctionData
	auctS := AuctionData{}
	err = protobuf.Decode(val, &auctS)
	require.Nil(t, err)

	return auctS
}

func printAuction(auction AuctionData) {
	fmt.Println("Seller account: ", auction.SellerAccount)
	fmt.Println("Good: ", auction.GoodDescription)
	fmt.Println("Reserve price: ", auction.ReservePrice)
	fmt.Println("State: ", auction.State.String())
	if auction.State == CLOSED {
		fmt.Println("Winner: ", auction.WinnerAccount)
	} else {
		if len(auction.Bids) == 0 {
			fmt.Println("Bids: none yet")
		} else {
			i := 1
			fmt.Println("Bids: ")
			for _, bid := range auction.Bids {
				fmt.Println("	Bidder", i, ": ", bid.BidderAccount)
				fmt.Println("		Bid:", bid.Bid)
				i++
			}
		}
	}

}
