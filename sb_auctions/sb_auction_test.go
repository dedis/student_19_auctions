package sb_auctions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.dedis.ch/cothority/v3/byzcoin"
)

func TestContractSBAuction_Spawn(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller and deposit account
	sellAccInstID := byzcoin.InstanceID{}
	depAccInstID := byzcoin.InstanceID{}

	//Creating auction
	good := "bananas"
	reservePrice := uint32(0)
	auctInstID, auctionData := bct.createAuction(t, sellAccInstID, depAccInstID, good, reservePrice)

	//Verify auction
	auctS := bct.verifCreateAuction(t, auctInstID, auctionData)
	printAuction(auctS)

	return
}

func TestContractSBAuction_Invoke(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller and deposit accounts
	sellAccInstID, depAccInstID := bct.createSellerAndDepositAccount(t)

	//Creating bidder account with amount
	amount := uint32(200)
	bidAccInstID := bct.createBidderAccount(t, amount)

	//Creating another bidder account with amount
	bidAccInstID2 := bct.createBidderAccount(t, amount)

	//Creating auction
	good := "bananas"
	reservePrice := uint32(0)
	auctInstID, auctionData := bct.createAuction(t, sellAccInstID, depAccInstID, good, reservePrice)

	//array of bids
	bids := []BidData{}

	//First bidder bids -> invoke bid
	bid := uint32(30)
	bidata, err := bct.createBid(t, auctInstID, bidAccInstID, bid)
	require.NoError(t, err)
	bids = append(bids, bidata)

	//Second bidder bids -> invoke bid
	bid = uint32(10)
	bidata, err = bct.createBid(t, auctInstID, bidAccInstID2, bid)
	require.NoError(t, err)
	bids = append(bids, bidata)

	auctS := bct.verifAddBidToAuction(t, auctInstID, auctionData, bids)

	//First bidder update bid
	bid = uint32(20)
	bidata, err = bct.createBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err, "cannot bid less than previous bid")

	//Second bidder update bid
	bid = uint32(40)
	bidata, err = bct.createBid(t, auctInstID, bidAccInstID2, bid)
	require.NoError(t, err)
	bids[1].Bid = bid

	auctS = bct.verifAddBidToAuction(t, auctInstID, auctionData, bids)

	printAuction(auctS)

	//test Close and Process
	err = bct.closeAuction(t, auctInstID)
	require.NoError(t, err)

	auctS = bct.verifCloseAuction(t, auctInstID)
	printAuction(auctS)

	//First bidder update bid
	bid = uint32(40)
	_, err = bct.createBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err, "auction is closed, cannot bid")

}
