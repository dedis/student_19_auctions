package auctions

import "go.dedis.ch/cothority/v3/byzcoin"

// PROTOSTART
// package auction;
//
// option java_package = "ch.epfl.dedis.template.proto";
// option java_outer_classname = "AuctionProto";

//Structures for an Auction instance

//Enum auction state
type state int

const (
	OPEN state = 1 + iota
	CLOSED
)

var states = [...]string{
	"OPEN",
	"CLOSED",
}

func (s state) String() string {
	return states[s-1]
}

type AuctionData struct {
	GoodDescription string
	SellerAccount   byzcoin.InstanceID // The place credit (transfer the coins to) when the auction is over
	//ReservePrice    uint64
	HighestBid BidData
	State      state // open or closed
	Deposit    byzcoin.InstanceID
}

type BidData struct {
	BidderAccount byzcoin.InstanceID // The place to refund if this bid is not accepted or debit if accepted.
	Bid           uint64
}
