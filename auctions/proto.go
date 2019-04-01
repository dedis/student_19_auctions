package auctions

import "go.dedis.ch/cothority/v3/byzcoin"

// PROTOSTART
// package auction;
//
// option java_package = "ch.epfl.dedis.template.proto";
// option java_outer_classname = "AuctionProto";

//Structures for an Auction instance

type state int

const (
	OPEN state = 1 + iota
	PROCESSING
	CLOSED
)

var states = [...]string{
	"OPEN",
	"PROCESSING",
	"CLOSED",
}

func (s state) String() string {
	return states[s-1]
}

type AuctionData struct {
	GoodDescription string
	SellerAccount   byzcoin.InstanceID // The place credit (transfer the coins to) when the auction is over
	ReservePrice    uint32
	Bids            []BidData
	State           state // open, processing or closed
	Deposits        byzcoin.InstanceID
	WinnerAccount   byzcoin.InstanceID
}

type BidData struct {
	BidderAccount byzcoin.InstanceID // The place to refund if this bid is not accepted or debit if accepted.
	prevBid       uint32             // serve also for previous bid
	Bid           uint32
}
