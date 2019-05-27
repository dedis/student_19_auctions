package centrilized_auctions

import (
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// Client is a structure to communicate with the template
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

// Bid chooses one server from the Roster at random. It
// sends a bid to it, which is then processed on the server side
// via the code in service.
//
// Bid will return the time in seconds it took to run the protocol.
func (c *Client) Bid(r *onet.Roster) (*BidReply, error) {
	dst := r.RandomServerIdentity()
	log.Lvl4("Sending message to", dst)
	reply := &BidReply{}
	err := c.SendProtobuf(dst, &Bid{r}, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// Close will return the number of times `Bid` has been called on this
// service-node = highestbid.
func (c *Client) Close(si *network.ServerIdentity) (int, error) {
	reply := &CloseReply{}
	err := c.SendProtobuf(si, &Close{}, reply)
	if err != nil {
		return -1, err
	}
	return reply.HighestBid, nil
}
