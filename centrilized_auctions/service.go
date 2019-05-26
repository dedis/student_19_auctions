package centrilized_auctions

import (
	"errors"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"sync"
)

// Used for tests
var centauctionID onet.ServiceID

func init() {
	var err error
	centauctionID, err = onet.RegisterNewService(ServiceName, newService)
	log.ErrFatal(err)
	network.RegisterMessage(&storage{})
}

// Service is our cent_auction-service
type Service struct {
	*onet.ServiceProcessor
	storage *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("cent_auction")

// storage is used to save our data.
type storage struct {
	HighestBid int
	sync.Mutex
}

// Bid starts a bid-protocol.
func (s *Service) Bid(r *Bid) (*BidReply, error) {
	s.storage.Lock()
	s.storage.HighestBid++
	s.storage.Unlock()
	s.save()
	tree := r.Roster.GenerateNaryTreeWithRoot(2, s.ServerIdentity())
	if tree == nil {
		return nil, errors.New("couldn't create tree")
	}
	pi, err := s.CreateProtocol(ProtocolName, tree)
	if err != nil {
		return nil, err
	}

	_ = pi.Start()
	resp := &BidReply{
		Children: <-pi.(*CentAuctionProtocol).ChildCount,
	}

	return resp, nil
}

// Count returns the number of instantiations of the protocol = highest bid.
func (s *Service) Close(arg *Close) (*CloseReply, error) {
	s.storage.Lock()
	defer s.storage.Unlock()
	return &CloseReply{HighestBid: s.storage.HighestBid}, nil
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolOnet, this will not be called, as the Onet will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *Service) NewProtocol(tn *onet.TreeNodeInstance, conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	log.Lvl3("Not templated yet")
	return nil, nil
}

// saves all data.
func (s *Service) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	s.storage = &storage{}
	msg, err := s.Load(storageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.storage, ok = msg.(*storage)
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
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
	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
