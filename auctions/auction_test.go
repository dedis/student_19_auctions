package auctions

import (
	"fmt"
	"testing"
	"time"

	"go.dedis.ch/protobuf"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
)

func TestContractAuction_Spawn(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	sellAccInstID := byzcoin.InstanceID{}

	auction := AuctionData{
		GoodDescription: "Bananas",
		SellerAccount:   sellAccInstID,
		Bids:            []BidData{},
		State:           "open",
	}

	auctionBuf, err := protobuf.Encode(&auction)
	if err != nil {
		return
	}

	// Try to spawn
	args := byzcoin.Arguments{
		{
			Name:  "auction",
			Value: auctionBuf,
		},
	}

	auctInstID := bct.createInstance(t, args)

	// Get the proof from byzcoin
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

	// Verify value.
	require.Equal(t, auction.GoodDescription, auctS.GoodDescription)
	require.Equal(t, auction.SellerAccount, auctS.SellerAccount)
	require.Equal(t, auction.State, auctS.State)

	fmt.Println(auctS)
	return
}

func TestContractAuction_Invoke(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	one := make([]byte, 32)
	one[31] = 1
	sellAccInstID := byzcoin.NewInstanceID(one)

	auction := AuctionData{
		GoodDescription: "Bananas",
		SellerAccount:   sellAccInstID,
		ReservePrice:    0,
		Bids:            []BidData{},
		State:           "open",
		WinnerAccount:   byzcoin.InstanceID{},
	}

	auctionBuf, err := protobuf.Encode(&auction)
	if err != nil {
		return
	}

	// Try to spawn
	auctionArgs := byzcoin.Arguments{
		{
			Name:  "auction",
			Value: auctionBuf,
		},
	}

	auctInstID := bct.createInstance(t, auctionArgs)

	two := make([]byte, 32)
	two[31] = 2
	bidAccInstID := byzcoin.NewInstanceID(two)

	bid := BidData{
		BidderAccount: bidAccInstID,
		Deposit:       0,
		Bid:           10,
	}

	bidBuf, err := protobuf.Encode(&bid)
	if err != nil {
		return
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

	require.Nil(t, ctx.FillSignersAndSignWith(bct.signer))
	_, err = bct.cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

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

	// Verify value.
	bids := []BidData{}
	bids = append(bids, bid)

	//fmt.Println(bids)

	require.Equal(t, auction.GoodDescription, auctS.GoodDescription)
	require.Equal(t, auction.SellerAccount, auctS.SellerAccount)
	require.Equal(t, auction.State, auctS.State)
	require.Equal(t, bids, auctS.Bids)

	//fmt.Println(auctS)
}

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
		[]string{"spawn:auction", "invoke:auction.bid"}, out.signer.Identity())
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
	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
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
	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
}
