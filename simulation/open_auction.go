package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dedis/student_19_auctions/auctions"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul/monitor"
	"go.dedis.ch/protobuf"
)

func init() {
	onet.SimulationRegister("OpenAuction", NewSimulationOpenAuction)
}

// SimulationService holds the state of the simulation.
type SimulationOpenAuction struct {
	onet.SimulationBFTree
	BlockInterval string
	Auctions      int
	Bidders       int
	Bids          int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationOpenAuction(config string) (onet.Simulation, error) {
	es := &SimulationOpenAuction{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationOpenAuction) Setup(dir string, hosts []string) (
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
func (s *SimulationOpenAuction) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationOpenAuction) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	signer := darc.NewSignerEd25519(nil, nil)

	// Create the ledger
	gm, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, config.Roster,
		[]string{"spawn:auction", "invoke:auction.bid", "invoke:auction.close", "invoke:auction.drop", "spawn:coin", "invoke:coin.mint", "invoke:coin.fetch"}, signer.Identity())
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
	coins := make([]byte, 8)
	binary.LittleEndian.PutUint64(coins, 100000)

	var instr []byzcoin.Instruction

	ct := uint64(1)
	for i := 0; i < s.Bidders; i++ {
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

	// Remember the bidder accounts
	bidderAccounts := make([]byzcoin.InstanceID, s.Bidders)
	for i := 0; i < s.Bidders; i++ {
		bidderAccounts[i] = tx.Instructions[i].DeriveID("")
	}

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

	// Create accounts for each seller
	instr = nil

	for i := 0; i < s.Auctions; i++ {
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

	// Remember the seller accounts
	sellerAccounts := make([]byzcoin.InstanceID, s.Auctions)
	for i := 0; i < s.Auctions; i++ {
		sellerAccounts[i] = tx.Instructions[i].DeriveID("")
		log.LLvl4(sellerAccounts[i])
	}

	// Create the auctions
	instID := byzcoin.InstanceID{}

	auction := auctions.AuctionData{
		GoodDescription: "bananas",
		HighestBid:      0,
		HighestBidder:   instID,
		State:           "OPEN",
		ReservePrice:    createHash("simulationsalt", 0),
	}

	instr = nil

	for i := 0; i < s.Auctions; i++ {

		auction.SellerAccount = sellerAccounts[i]

		auctionBuf, err := protobuf.Encode(&auction)
		if err != nil {
			return err
		}

		spauct := byzcoin.Instruction{
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
		}
		instr = append(instr, spauct)
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
	log.Lvlf1("Spawn auctions")
	send := monitor.NewTimeMeasure("send")
	_, err = c.AddTransactionAndWait(tx, 2)
	if err != nil {
		return errors.New("couldn't initialize accounts: " + err.Error())
	}

	// Remember the auctions IDs
	auctionIDs := make([]byzcoin.InstanceID, s.Auctions)
	for i := 0; i < s.Auctions; i++ {
		auctionIDs[i] = tx.Instructions[i].DeriveID("")
	}
	send.Record()

	// For each auction, have the senders send bids, close the auction
	// and confirm that the auction result was correct.

	amount := make([]byte, 8)
	bidamount := uint64(0)

	bidata := auctions.BidData{
		Bid: 0,
	}

	for round := 0; round < s.Bids; round++ {
		log.Lvl1("Starting round", round)
		roundM := monitor.NewTimeMeasure("round")

		for loop1 := 0; loop1 < s.Bidders; loop1++ {

			instr = nil
			bidamount = bidamount + uint64(1)
			binary.LittleEndian.PutUint64(amount, uint64(bidamount))

			for loop2 := 0; loop2 < s.Auctions; loop2++ {

				bidata.BidderAccount = bidderAccounts[loop1]
				bidata.BidderPubKey = bidderAccounts[loop1].String()

				bidBuf, err := protobuf.Encode(&bidata)
				if err != nil {
					return err
				}

				fetch := byzcoin.Instruction{
					InstanceID: bidderAccounts[loop1],
					Invoke: &byzcoin.Invoke{
						ContractID: contracts.ContractCoinID,
						Command:    "fetch",
						Args: byzcoin.Arguments{{
							Name:  "coins",
							Value: amount}},
					},
					SignerIdentities: []darc.Identity{signer.Identity()},
					SignerCounter:    []uint64{ct},
				}

				bid := byzcoin.Instruction{
					InstanceID: auctionIDs[loop2],
					Invoke: &byzcoin.Invoke{
						ContractID: auctions.ContractAuctionID,
						Command:    "bid",
						Args: byzcoin.Arguments{{
							Name:  "bid",
							Value: bidBuf}},
					},
					SignerIdentities: []darc.Identity{signer.Identity()},
					SignerCounter:    []uint64{ct + 1},
				}

				instr = nil
				instr = append(instr, fetch)
				instr = append(instr, bid)
				ct += 2

				// sign instruction
				tx = byzcoin.ClientTransaction{
					Instructions: instr,
				}
				if err = tx.FillSignersAndSignWith(signer); err != nil {
					return errors.New("signing of instruction failed: " + err.Error())
				}
				// Send the instructions.
				_, err = c.AddTransactionAndWait(tx, 10)
				if err != nil {
					return errors.New("couldn't bid: " + err.Error())
				}

			}
		}

		roundM.Record()

		// This sleep is needed to wait for the propagation to finish
		// on all the nodes. Otherwise the simulation manager
		// (runsimul.go in onet) might close some nodes and cause
		// skipblock propagation to fail.
		time.Sleep(blockInterval)
	}

	// Close the auctions. Send in that transaction with AddTxAndWait,
	// and then check the final value of the highest bidder and that
	// the coins have been transfered into the sellerAccount.

	confirm := monitor.NewTimeMeasure("confirm")

	// tx := ...an auction close tx
	closedata := auctions.CloseData{
		Salt:         "simulationsalt",
		ReservePrice: 0,
	}

	closeBuf, err := protobuf.Encode(&closedata)
	if err != nil {
		return err
	}

	instr = nil

	for loop2 := 0; loop2 < s.Auctions; loop2++ {
		closing := byzcoin.Instruction{
			InstanceID: auctionIDs[loop2],
			Invoke: &byzcoin.Invoke{
				ContractID: auctions.ContractAuctionID,
				Command:    "close",
				Args: byzcoin.Arguments{{
					Name:  "close",
					Value: closeBuf}},
			},
			SignerIdentities: []darc.Identity{signer.Identity()},
			SignerCounter:    []uint64{ct},
		}

		instr = append(instr, closing)
		ct += 1

	}

	// sign instruction
	tx = byzcoin.ClientTransaction{
		Instructions: instr,
	}
	if err = tx.FillSignersAndSignWith(signer); err != nil {
		return errors.New("signing of instruction failed: " + err.Error())
	}
	// Send the instructions.
	_, err = c.AddTransactionAndWait(tx, 20)
	if err != nil {
		return errors.New("couldn't close auction: " + err.Error())
	}

	var account byzcoin.Coin
	for loop2 := 0; loop2 < s.Auctions; loop2++ {

		proof, err := c.GetProof(sellerAccounts[loop2].Slice())
		if err != nil {
			return errors.New("couldn't get proof for transaction: " + err.Error())
		}
		_, v0, _, _, err := proof.Proof.KeyValue()
		if err != nil {
			return errors.New("proof doesn't hold transaction: " + err.Error())
		}

		err = protobuf.Decode(v0, &account)
		if err != nil {
			return errors.New("couldn't decode account: " + err.Error())
		}

		log.Lvlf1("Account has %d", account.Value)
		if account.Value != uint64(s.Bidders*s.Bids) {
			log.LLvl4("seller account at end", account.Value)
			return errors.New("account has wrong amount")
		}
	}

	confirm.Record()

	// This sleep is needed to wait for the propagation to finish
	// on all the nodes. Otherwise the simulation manager
	// (runsimul.go in onet) might close some nodes and cause
	// skipblock propagation to fail.
	time.Sleep(blockInterval)

	return nil
}

func createHash(salt string, reservP uint64) string {
	strReservePrice := strconv.Itoa(int(reservP))
	h := sha256.New()
	h.Write([]byte(salt + strReservePrice))
	hashed := h.Sum(nil)
	hash := hex.EncodeToString(hashed)
	return hash
}
