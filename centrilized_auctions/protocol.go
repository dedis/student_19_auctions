package centrilized_auctions

import (
	"errors"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

// CentAuctionProtocol holds the state of the centrilized auction protocol.
//
// For this example, it defines a channel that will receive the number
// of children. Only the root-node will write to the channel.
type CentAuctionProtocol struct {
	*onet.TreeNodeInstance
	ChildCount chan int
}

// Check that *TemplateProtocol implements onet.ProtocolInstance
var _ onet.ProtocolInstance = (*CentAuctionProtocol)(nil)

// NewProtocol initialises the structure for use in one round
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	t := &CentAuctionProtocol{
		TreeNodeInstance: n,
		ChildCount:       make(chan int),
	}
	for _, handler := range []interface{}{t.HandleAnnounce, t.HandleReply} {
		if err := t.RegisterHandler(handler); err != nil {
			return nil, errors.New("couldn't register handler: " + err.Error())
		}
	}
	return t, nil
}

// Start sends the Announce-message "new bid" to all children
func (p *CentAuctionProtocol) Start() error {
	log.Lvl3("Starting CentAuctionProtocol")
	return p.HandleAnnounce(StructNewBid{p.TreeNode(),
		NewBid{"new bid!"}})
}

// HandleAnnounce is the first message and is used to send an ID that
// is stored in all nodes.
func (p *CentAuctionProtocol) HandleAnnounce(msg StructNewBid) error {
	log.Lvl3("Parent announces:", msg.Message)
	if !p.IsLeaf() {
		// If we have children, send the same message to all of them
		_ = p.SendToChildren(&msg.NewBid)
	} else {
		// If we're the leaf, start to reply
		_ = p.HandleReply(nil)
	}
	return nil
}

// HandleReply is the message going up the tree and holding a counter
// to verify the number of nodes.
func (p *CentAuctionProtocol) HandleReply(reply []StructReply) error {
	defer p.Done()

	children := 1
	for _, c := range reply {
		children += c.ChildrenCount
	}
	log.Lvl3(p.ServerIdentity().Address, "is done with total of", children)

	if !p.IsRoot() {
		log.Lvl3("Sending to parent")
		return p.SendTo(p.Parent(), &Reply{children})
	}

	log.Lvl3("Root-node is done - nbr of children found:", children)
	p.ChildCount <- children
	return nil
}
