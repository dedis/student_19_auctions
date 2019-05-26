package centrilized_auctions

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// Name can be used from other packages to refer to this protocol.
const ServiceName = "centrilized_auctions"

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		Bid{}, BidReply{},
		Close{}, CloseReply{},
	)
}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

// Bid will run the centauction-protocol on the roster
type Bid struct {
	Roster *onet.Roster
}

// BidReply returns the number of children
type BidReply struct {
	Children int
}

// Close returns the number of protocol-runs = highest bid
type Close struct {
	Roster *onet.Roster
}

type CloseReply struct {
	HighestBid int
}
