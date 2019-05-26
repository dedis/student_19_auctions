package centrilized_auctions

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// ProtocolName can be used from other packages to refer to this protocol.
const ProtocolName = "CentAuction"

func init() {
	network.RegisterMessage(NewBid{})
	network.RegisterMessage(Reply{})
	_, _ = onet.GlobalProtocolRegister(ProtocolName, NewProtocol)
}

// NewBid is used to pass a message to all children.
type NewBid struct {
	Message string
}

// StructAnnounce just contains Announce and the data necessary to identify and
// process the message in the sda framework.
type StructNewBid struct {
	*onet.TreeNode
	NewBid
}

// Reply returns the count of all children.
type Reply struct {
	ChildrenCount int
}

// StructReply just contains Reply and the data necessary to identify and
// process the message in the sda framework.
type StructReply struct {
	*onet.TreeNode
	Reply
}
