package auctions

import (
	"go.dedis.ch/cothority/v3/byzcoin"
)

// PROTOSTART
// package auctions;
// import "byzcoin.proto";
//
// option java_package = "ch.epfl.dedis.lib.proto";
// option java_outer_classname = "Auctions";

// Auction struct

type AuctionData struct {
	GoodDescription string
	SellerAccount   byzcoin.InstanceID
	ReservePrice    string `protobuf:"opt"`
	HighestBid      uint64
	HighestBidder   byzcoin.InstanceID
	State           string
	WinProof        string
}

type BidData struct {
	BidderAccount byzcoin.InstanceID
	BidderPubKey  string
	Bid           uint64
}

type CloseData struct {
	Salt         string
	ReservePrice uint64
}
