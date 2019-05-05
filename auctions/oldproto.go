package auctions

//import "go.dedis.ch/cothority/v3/byzcoin"
//
//// PROTOSTART
//// package byzcoin;
//// type :skipchain.SkipBlockID:bytes
//// type :darc.ID:bytes
//// type :darc.Action:string
//// type :Arguments:[]Argument
//// type :Instructions:[]Instruction
//// type :TxResults:[]TxResult
//// type :InstanceID:bytes
//// type :Version:sint32
//// import "skipchain.proto";
//// import "onet.proto";
//// import "darc.proto";
//// import "trie.proto";
////
//// option java_package = "ch.epfl.dedis.lib.proto";
//// option java_outer_classname = "ByzCoinProto";
//
//
//type state int
//
//const (
//	OPEN state = 1 + iota
//	CLOSED
//)
//
//var states = [...]string{
//	"OPEN",
//	"CLOSED",
//}
//
//func (s state) String() string {
//	return states[s-1]
//}
//
//// Structures for an Auction instance
//
//type AuctionData struct {
//	GoodDescription string
//	SellerAccount   byzcoin.InstanceID // The place credit (transfer the coins to) when the auction is over
//	//ReservePrice    uint64
//	HighestBid BidData
//	State      state // open or closed
//	//Deposit    byzcoin.InstanceID
//}
//
//type BidData struct {
//	BidderAccount byzcoin.InstanceID // The place to refund if this bid is not accepted or debit if accepted.
//	Bid           uint64
//}
