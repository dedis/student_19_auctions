package centrilized_auctions

import (
	"errors"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

func init() {
	var err error
	_, err = onet.RegisterNewService(ServiceName, newService)
	log.ErrFatal(err)
}

// Service is our cent_auction-service
type Service struct {
	*onet.ServiceProcessor
	HighestBid int
}

func (s *Service) Bid(r *Bid) (*BidReply, error) {
	if s.HighestBid >= r.Bid {
		return nil, errors.New("bid too low")
	}
	s.HighestBid = r.Bid
	return &BidReply{}, nil
}

// Count returns the highest bid.
func (s *Service) Close(arg *Close) (*CloseReply, error) {
	reply := &CloseReply{HighestBid: s.HighestBid}
	s.HighestBid = 0
	return reply, nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	if err := s.RegisterHandlers(s.Bid, s.Close); err != nil {
		// return nil, errors.New("Couldn't register messages")
		return nil, err
	}
	return s, nil
}
