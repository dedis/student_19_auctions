package main_test

import (
	"go.dedis.ch/onet/v3/simul"
	"testing"

	"go.dedis.ch/onet/v3/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func Test_Open(t *testing.T) {
	simul.Start("open_auction.toml")
}

func Test_Centralized(t *testing.T) {
	simul.Start("centrilized_auction.toml")
}
