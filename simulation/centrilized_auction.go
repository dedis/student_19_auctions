package main

import (
	"github.com/BurntSushi/toml"
	"github.com/dedis/student_19_auctions/centrilized_auctions"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul/monitor"
)

func init() {
	onet.SimulationRegister("CentrilizedAuction", NewSimulationCentAuction)
}

// SimulationService holds the state of the simulation.
type SimulationCentAuction struct {
	onet.SimulationBFTree
	BlockInterval string
	Auctions      int
	Bidders       int
	Bids          int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationCentAuction(config string) (onet.Simulation, error) {
	es := &SimulationCentAuction{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationCentAuction) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationCentAuction) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationCentAuction) Run(config *onet.SimulationConfig) error {

	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)

	c := centrilized_auctions.NewClient()
	bid := 0

	for round := 0; round < s.Bids; round++ {
		log.Lvl1("Starting round", round)
		round := monitor.NewTimeMeasure("round")

		for loop1 := 0; loop1 < s.Bidders; loop1++ {
			for loop2 := 0; loop2 < s.Auctions; loop2++ {
				bid++
				_, err := c.Bid(config.Roster, bid)
				log.ErrFatal(err)
			}
		}

		round.Record()
	}

	high, err := c.Close(config.Roster.List[0])
	if high != bid {
		log.Error("wrong high bid")
	}
	log.ErrFatal(err)

	return nil
}
