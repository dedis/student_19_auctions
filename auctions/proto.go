package auctions

import (
	"go.dedis.ch/cothority/v3/byzcoin"
)

// PROTOSTART
// package auctions;
// type :state:sint64
// import "byzcoin.proto";
//
// option java_package = "ch.epfl.dedis.lib.proto";
// option java_outer_classname = "Auction";

// Auction struct

type AuctionData struct {
	GoodDescription string
	SellerAccount   byzcoin.InstanceID
	//InitialPrice    uint64
	HighestBid BidData
	State      state
}

type BidData struct {
	BidderAccount byzcoin.InstanceID
	Bid           uint64
}
