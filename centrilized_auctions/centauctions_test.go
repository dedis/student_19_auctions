package centrilized_auctions

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"testing"
	"time"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestClient_Bid(t *testing.T) {
	nbr := 5
	local := onet.NewTCPTest(tSuite)
	_, roster, _ := local.GenTree(nbr, true)
	defer local.CloseAll()

	c := NewClient()
	cl1, err := c.Bid(roster)
	require.Nil(t, err)
	require.Equal(t, nbr, cl1.Children)
	cl2, err := c.Bid(roster)
	require.Nil(t, err)
	require.Equal(t, nbr, cl2.Children)
}

func TestClient_Close(t *testing.T) {

	local := onet.NewTCPTest(tSuite)

	nbr := 5

	_, roster, _ := local.GenTree(nbr, true)
	defer local.CloseAll()

	c := NewClient()

	// Verify there is no bid
	for _, s := range roster.List {
		highestbid, err := c.Close(s)
		require.Nil(t, err)
		require.Equal(t, 0, highestbid)
	}

	nbBidders := 6
	// Make some bid-requests
	for i := 0; i < nbBidders; i++ {
		_, err := c.Bid(roster)
		require.Nil(t, err)
	}

	// Verify we have the correct total of bid
	highestbid := 0
	for _, s := range roster.List {
		highb, err := c.Close(s)
		require.Nil(t, err)
		highestbid = highb
	}
	require.Equal(t, nbBidders, highestbid)
}

// Tests a 2, 5 and 13-node system. It is good practice to test different
// sizes of trees to make sure your protocol is stable.
func TestProtocol(t *testing.T) {
	nodes := []int{2, 5, 13}
	for _, nbrNodes := range nodes {
		local := onet.NewLocalTest(tSuite)
		_, _, tree := local.GenTree(nbrNodes, true)
		log.Lvl3(tree.Dump())

		pi, err := local.StartProtocol(ProtocolName, tree)
		require.Nil(t, err)
		protocol := pi.(*CentAuctionProtocol)
		timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
		select {
		case children := <-protocol.ChildCount:
			log.Lvl2("Instance 1 is done")
			require.Equal(t, children, nbrNodes, "Didn't get a child-cound of", nbrNodes)
		case <-time.After(timeout):
			t.Fatal("Didn't finish in time")
		}
		local.CloseAll()
	}
}

func TestService_Bid(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, roster, _ := local.GenTree(5, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, centauctionID)

	for _, s := range services {
		log.Lvl2("Sending request to", s)
		resp, err := s.(*Service).Bid(
			&Bid{Roster: roster},
		)
		require.Nil(t, err)
		require.Equal(t, resp.Children, len(roster.List))
	}
}

func TestService_Close(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, roster, _ := local.GenTree(5, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, centauctionID)

	for _, s := range services {
		log.Lvl2("Sending request to", s)
		resp, err := s.(*Service).Bid(
			&Bid{Roster: roster},
		)
		require.Nil(t, err)
		require.Equal(t, resp.Children, len(roster.List))
		reply, err := s.(*Service).Close(&Close{})
		require.Nil(t, err)
		require.Equal(t, 1, reply.HighestBid)
	}
}
