package byzcoin

// PROTOSTART
// package auction;
//
// option java_package = "ch.epfl.dedis.template.proto";
// option java_outer_classname = "AuctionProto";

//Structures for an Auction instance

type state string 

type AuctionData struct {
	GoodDescription string
	SellerAccount byzcoin.InstanceID // The place credit (transfer the coins to) when the auction is over
	//ReservePrice unint32 
	Bids []BidData
	State state // open, calculating or closed
}

type BidData struct {
	BidderAccount byzcoin.InstanceID // The place to refund if this bid is not accepted or debit if accepted.
	Deposit uint32
	Bid uint32
}
