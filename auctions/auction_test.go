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

func TestContractAuction_Invoke(t *testing.T) {
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

	//First bidder bids -> invoke bid
	bid := uint32(10)
	bidata, err := bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.NoError(t, err)

	//Second bidder bids -> invoke bid
	bid = uint32(30)
	bidata, err = bct.addBid(t, auctInstID, bidAccInstID2, bid)
	require.NoError(t, err)

	auctS := bct.verifAddBid(t, auctInstID, auctionData, bidata)
	printAuction(auctS)

	//First bidder update bid
	bid = uint32(20)
	_, err = bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err, "cannot bid less than current highest bid")

	auctS = bct.verifAddBid(t, auctInstID, auctionData, bidata)
	printAuction(auctS)

	//Close auction
	bid = uint32(20)
	err = bct.closeAuction(t, auctInstID)
	require.NoError(t, err)

	auctS = bct.verifCloseAuction(t, auctInstID)
	printAuction(auctS)

	//First bidder update bid
	bid = uint32(40)
	_, err = bct.addBid(t, auctInstID, bidAccInstID, bid)
	require.Error(t, err, "auction is closed, cannot bid")

}
