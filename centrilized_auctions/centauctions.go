package centrilized_auctions

import (
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// Client is a structure to communicate with the centrilized auction
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

func (c *Client) Bid(r *onet.Roster, bid int) (*BidReply, error) {
	dst := r.RandomServerIdentity()
	log.Lvl4("Sending message to", dst)
	reply := &BidReply{}
	err := c.SendProtobuf(dst, &Bid{bid}, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) Close(si *network.ServerIdentity) (int, error) {
	reply := &CloseReply{}
	err := c.SendProtobuf(si, &Close{}, reply)
	if err != nil {
		return -1, err
	}
	return reply.HighestBid, nil
}
