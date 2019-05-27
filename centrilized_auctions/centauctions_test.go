package centrilized_auctions

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"testing"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestClient_Bid(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	_, roster, _ := local.GenTree(1, true)
	defer local.CloseAll()

	c := NewClient()
	_, err := c.Bid(roster, 0)
	require.Error(t, err)
	_, err = c.Bid(roster, 1)
	require.Nil(t, err)
	_, err = c.Bid(roster, 1)
	require.Error(t, err)
}

func TestClient_Close(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	_, roster, _ := local.GenTree(1, true)
	defer local.CloseAll()

	c := NewClient()

	// Verify there is no bid
	highestbid, err := c.Close(roster.List[0])
	require.Nil(t, err)
	require.Equal(t, 0, highestbid)

	nbBidders := 6
	// Make some bid-requests
	for i := 0; i < nbBidders; i++ {
		_, err := c.Bid(roster, i+1)
		require.Nil(t, err)
	}

	// Verify we have the correct total of bid
	highb, err := c.Close(roster.List[0])
	require.Nil(t, err)
	require.Equal(t, nbBidders, highb)
}
