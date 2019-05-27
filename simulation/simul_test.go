package main_test

import (
	"go.dedis.ch/onet/v3/simul"
	"testing"

	"go.dedis.ch/onet/v3/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestSimulation(t *testing.T) {
	//simul.Start("open_auction.toml", "centrilized_auction.toml")
	simul.Start("open_auction.toml")
}
