package main

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dedis/student_19_auctions/auctions"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul"
	"go.dedis.ch/onet/v3/simul/monitor"
	"go.dedis.ch/protobuf"
)

func main() {
	simul.Start()
}

func init() {
	onet.SimulationRegister("Auction", NewSimulationService)
}

// SimulationService holds the state of the simulation.
type SimulationService struct {
	onet.SimulationBFTree
	BlockInterval string
	Bidders       int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (
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
func (s *SimulationService) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	signer := darc.NewSignerEd25519(nil, nil)

	// Create the ledger
	gm, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, config.Roster,
		[]string{"spawn:auction", "invoke:auction.bid", "invoke:auction.close", "spawn:coin", "invoke:coin.mint", "invoke:coin.fetch"}, signer.Identity())
	if err != nil {
		return errors.New("couldn't setup genesis message: " + err.Error())
	}

	// Set block interval from the simulation config.
	blockInterval, err := time.ParseDuration(s.BlockInterval)
	if err != nil {
		return errors.New("parse duration of BlockInterval failed: " + err.Error())
	}
	gm.BlockInterval = blockInterval

	c, _, err := byzcoin.NewLedger(gm, false)
	if err != nil {
		return errors.New("couldn't create genesis block: " + err.Error())
	}

	// Create accounts for each bidder and give them 1000 coins to use.
	// Create one extra account for the seller.
	coins := make([]byte, 8)
	binary.LittleEndian.PutUint64(coins, 1000)

	var instr []byzcoin.Instruction

	ct := uint64(1)
	for i := 0; i < s.Bidders+1; i++ {
		acct := byzcoin.Instruction{
			InstanceID: byzcoin.NewInstanceID(gm.GenesisDarc.GetBaseID()),
			Spawn: &byzcoin.Spawn{
				ContractID: contracts.ContractCoinID,
			},
			SignerIdentities: []darc.Identity{signer.Identity()},
			SignerCounter:    []uint64{ct},
		}
		instr = append(instr, acct)
		ct += 1
	}

	// Now sign all the instructions
	tx := byzcoin.ClientTransaction{
		Instructions: instr,
	}
	if err = tx.FillSignersAndSignWith(signer); err != nil {
		return errors.New("signing of instruction failed: " + err.Error())
	}
	// Send the instructions.
	_, err = c.AddTransactionAndWait(tx, 2)
	if err != nil {
		return errors.New("couldn't initialize accounts: " + err.Error())
	}

	// Remember the bidder accounts and the seller account (the last one)
	bidderAccounts := make([]byzcoin.InstanceID, s.Bidders)
	for i := 0; i < s.Bidders; i++ {
		bidderAccounts[i] = tx.Instructions[i].DeriveID("")
	}
	sellerAccount := tx.Instructions[s.Bidders].DeriveID("")

	// Now put coins in all the bidder accounts.
	instr = nil
	for i := 0; i < s.Bidders; i++ {
		mint := byzcoin.Instruction{
			InstanceID: bidderAccounts[i],
			Invoke: &byzcoin.Invoke{
				ContractID: contracts.ContractCoinID,
				Command:    "mint",
				Args: byzcoin.Arguments{{
					Name:  "coins",
					Value: coins}},
			},
			SignerIdentities: []darc.Identity{signer.Identity()},
			SignerCounter:    []uint64{ct},
		}
		instr = append(instr, mint)
		ct += 1
	}
	// Now sign all the instructions
	tx = byzcoin.ClientTransaction{
		Instructions: instr,
	}
	if err = tx.FillSignersAndSignWith(signer); err != nil {
		return errors.New("signing of instruction failed: " + err.Error())
	}
	// Send the instructions.
	_, err = c.AddTransactionAndWait(tx, 2)
	if err != nil {
		return errors.New("couldn't initialize accounts: " + err.Error())
	}

	// For each round, open an auction, have the senders send bids, close the auction
	// and confirm that the auction result was correct.

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		roundM := monitor.NewTimeMeasure("round")

		// Open auction
		auction := auctions.AuctionData{
			GoodDescription: "Bananas",
			SellerAccount:   sellerAccount,
			State:           "OPEN",
		}

		auctionBuf, err := protobuf.Encode(&auction)
		if err != nil {
			return err
		}

		tx := byzcoin.ClientTransaction{
			Instructions: []byzcoin.Instruction{
				{
					InstanceID: byzcoin.NewInstanceID(gm.GenesisDarc.GetBaseID()),
					Spawn: &byzcoin.Spawn{
						ContractID: auctions.ContractAuctionID,
						Args: byzcoin.Arguments{
							{
								Name:  "auction",
								Value: auctionBuf,
							},
						},
					},
					SignerIdentities: []darc.Identity{signer.Identity()},
					SignerCounter:    []uint64{ct},
				},
			},
		}
		ct++

		if err = tx.FillSignersAndSignWith(signer); err != nil {
			return errors.New("signing of instruction failed: " + err.Error())
		}

		log.Lvlf1("Spawn auction")
		send := monitor.NewTimeMeasure("send")
		_, err = c.AddTransaction(tx)
		if err != nil {
			return errors.New("couldn't add transfer transaction: " + err.Error())
		}
		send.Record()

		// Now write two loops from 0 to s.Bidders and from 0 to s.Bids
		// and send in bids.

		// Close the auction. Send in that transaction with AddTxAndWait,
		// and then check the final value of the highest bidder and that
		// the coins have been transfered into the sellerAccount.

		confirm := monitor.NewTimeMeasure("confirm")

		// tx := ...an auction close tx
		// _, err = c.AddTransactionAndWait(tx, 20)
		// if err != nil {
		// 	return errors.New("while adding transaction and waiting: " + err.Error())
		// }

		proof, err := c.GetProof(sellerAccount.Slice())
		if err != nil {
			return errors.New("couldn't get proof for transaction: " + err.Error())
		}
		_, v0, _, _, err := proof.Proof.KeyValue()
		if err != nil {
			return errors.New("proof doesn't hold transaction: " + err.Error())
		}
		var account byzcoin.Coin
		err = protobuf.Decode(v0, &account)
		if err != nil {
			return errors.New("couldn't decode account: " + err.Error())
		}
		log.Lvlf1("Account has %d", account.Value)
		if account.Value != uint64(1000) {
			return errors.New("account has wrong amount")
		}
		confirm.Record()

		roundM.Record()

		// This sleep is needed to wait for the propagation to finish
		// on all the nodes. Otherwise the simulation manager
		// (runsimul.go in onet) might close some nodes and cause
		// skipblock propagation to fail.
		time.Sleep(blockInterval)
	}

	// We wait a bit before closing because c.GetProof is sent to the
	// leader, but at this point some of the children might still be doing
	// updateCollection. If we stop the simulation immediately, then the
	// database gets closed and updateCollection on the children fails to
	// complete.
	time.Sleep(time.Second)

	return nil
}
