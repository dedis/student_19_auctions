package auctions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.dedis.ch/cothority/v3/byzcoin"
)

func TestContractAuction_Spawn(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller and deposit account
	sellAccInstID := byzcoin.InstanceID{}

	//Creating auction
	good := "bananas"
	//reservePrice := uint64(0)
	auctInstID, auctionData := bct.createAuction(t, sellAccInstID, good)

	//Verify auction
	auctS := bct.verifCreateAuction(t, auctInstID, auctionData)
	printAuction(auctS)

	return
}

func TestContractAuction_Invoke(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller and deposit accounts
	sellAccInstID := bct.createSellerAccount(t)

	//Creating bidder account with amount
	amount := uint64(200)
	bidAccInstID := bct.createBidderAccount(t, amount)

	//Creating another bidder account with amount
	bidAccInstID2 := bct.createBidderAccount(t, amount)

	//Creating auction
	good := "bananas"
	//reservePrice := uint64(0)
	auctInstID, auctionData := bct.createAuction(t, sellAccInstID, good)

	//First bidder bids -> invoke bid
	bid := uint64(20)
	bidata, err := bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.NoError(t, err)

	//Second bidder bids -> invoke bid
	bid = uint64(40)
	bidata, err = bct.addBid(t, auctInstID, bidAccInstID2, bid)
	require.NoError(t, err)

	bidata.Bid = bid
	auctS := bct.verifAddBid(t, auctInstID, auctionData, bidata)
	printAuction(auctS)

	//First bidder update bid
	bid = uint64(40)
	_, err = bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err)

	auctS = bct.verifAddBid(t, auctInstID, auctionData, bidata)
	printAuction(auctS)

	//Close auction
	err = bct.closeAuction(t, auctInstID)
	require.NoError(t, err)

	auctS = bct.verifCloseAuction(t, auctInstID)
	printAuction(auctS)

	//First bidder update bid
	bid = uint64(40)
	_, err = bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err, "auction is closed, cannot bid")

}

func TestContractAuction_Invoke2(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller and deposit accounts
	sellAccInstID := bct.createSellerAccount(t)

	//Creating bidder account with amount
	amount := uint64(200)
	bidAccInstID := bct.createBidderAccount(t, amount)

	//Creating auction
	good := "bananas"
	//reservePrice := uint64(0)
	auctInstID, _ := bct.createAuction(t, sellAccInstID, good)

	//First bidder bids with 0 coins
	bid := uint64(0)
	_, err := bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err)

	//Close auction
	err = bct.closeAuction(t, auctInstID)
	require.NoError(t, err)

	auctS := bct.verifCloseAuction(t, auctInstID)
	printAuction(auctS)

}

func TestContractAuction_Invoke3(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	//Creating seller account with amount
	amount := uint64(200)
	sellAccInstID := bct.createBidderAccount(t, amount)

	//Creating auction
	good := "bananas"
	//reservePrice := uint64(0)
	auctInstID, _ := bct.createAuction(t, sellAccInstID, good)

	//First bidder bids -> invoke bid
	bid := uint64(20)
	_, err := bct.addBid(t, auctInstID, sellAccInstID, bid)
	require.Error(t, err)

	//Close auction
	err = bct.closeAuction(t, auctInstID)
	require.NoError(t, err)

	auctS := bct.verifCloseAuction(t, auctInstID)
	printAuction(auctS)

}
