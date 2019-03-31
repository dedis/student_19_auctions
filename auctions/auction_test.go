package auctions

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.dedis.ch/protobuf"

	"go.dedis.ch/cothority/v3/byzcoin/trie"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/onet/v3/log"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
)

func TestSpawn(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	amount := make([]byte, 8)
	binary.LittleEndian.PutUint32(amount, 300)

	//account := contractAccount{}
	//sellAccInstID, erro := account.createAccount(amount)
	//if erro != nil {
	//	fmt.Println("Couldn't create seller account")
	//	return
	//}

	sellAccInstID := byzcoin.InstanceID{}

	auction := AuctionData{
		GoodDescription: "Bananas",
		SellerAccount:   sellAccInstID,
		Bids:            []BidData{},
		State:           "open",
	}

	auctionBuf, err := protobuf.Encode(&auction)
	if err != nil {
		return
	}

	// Try to spawn
	args := byzcoin.Arguments{
		{
			Name:  "auction",
			Value: auctionBuf,
		},
	}

	auctInstID := bct.createInstance(t, args)

	// Get the proof from byzcoin
	reply, err := bct.cl.GetProof(auctInstID.Slice())
	require.Nil(t, err)
	// Make sure the proof is a matching proof and not a proof of absence.
	proof := reply.Proof
	require.True(t, proof.InclusionProof.Match(auctInstID.Slice()))

	// Get the raw values of the proof.
	_, val, _, _, err := proof.KeyValue()
	require.Nil(t, err)

	// And decode the buffer to a AuctionData
	auctS := AuctionData{}
	err = protobuf.Decode(val, &auctS)
	require.Nil(t, err)

	// Verify value.
	require.Equal(t, auction.GoodDescription, auctS.GoodDescription)
	require.Equal(t, auction.SellerAccount, auctS.SellerAccount)
	require.Equal(t, auction.State, auctS.State)

	fmt.Println(auctS)

	return
}

// bcTest is used here to provide some simple test structure for different
// tests.
type bcTest struct {
	local   *onet.LocalTest
	signer  darc.Signer
	servers []*onet.Server
	roster  *onet.Roster
	cl      *byzcoin.Client
	gMsg    *byzcoin.CreateGenesisBlock
	gDarc   *darc.Darc
	ct      uint64
}

func newBCTest(t *testing.T) (out *bcTest) {
	out = &bcTest{}
	// First create a local test environment with three nodes.
	out.local = onet.NewTCPTest(cothority.Suite)

	out.signer = darc.NewSignerEd25519(nil, nil)
	out.servers, out.roster, _ = out.local.GenTree(3, true)

	// Then create a new ledger with the genesis darc having the right
	// to create and update keyValue contracts.
	var err error
	out.gMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, out.roster,
		[]string{"spawn:auction", "invoke:keyValue.update"}, out.signer.Identity())
	require.Nil(t, err)
	out.gDarc = &out.gMsg.GenesisDarc

	// This BlockInterval is good for testing, but in real world applications this
	// should be more like 5 seconds.
	out.gMsg.BlockInterval = time.Second / 2

	out.cl, _, err = byzcoin.NewLedger(out.gMsg, false)
	require.Nil(t, err)
	out.ct = 1

	return out
}

func (bct *bcTest) Close() {
	bct.local.CloseAll()
}

func (bct *bcTest) createInstance(t *testing.T, args byzcoin.Arguments) byzcoin.InstanceID {
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID:    byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
			SignerCounter: []uint64{bct.ct},
			Spawn: &byzcoin.Spawn{
				ContractID: ContractAuctionID,
				Args:       args,
			},
		}},
	}
	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
	return ctx.Instructions[0].DeriveID("")
}

func (bct *bcTest) updateInstance(t *testing.T, instID byzcoin.InstanceID, args byzcoin.Arguments) {
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID:    instID,
			SignerCounter: []uint64{bct.ct},
			Invoke: &byzcoin.Invoke{
				ContractID: ContractAuctionID,
				Command:    "update",
				Args:       args,
			},
		}},
	}
	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	var err error
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
}

// Used to create the accounts
var gdarc *darc.Darc
var gsigner darc.Signer

type contractAccount struct {
	byzcoin.BasicContract
	byzcoin.Coin
	darc.Darc
	s *byzcoin.Service
}

type cvTest struct {
	values      map[string][]byte
	contractIDs map[string]string
	darcIDs     map[string]darc.ID
	index       int
}

func newCT(rStr ...string) *cvTest {
	ct := &cvTest{
		make(map[string][]byte),
		make(map[string]string),
		make(map[string]darc.ID),
		0,
	}
	gsigner = darc.NewSignerEd25519(nil, nil)
	rules := darc.InitRules([]darc.Identity{gsigner.Identity()},
		[]darc.Identity{gsigner.Identity()})
	for _, r := range rStr {
		rules.AddRule(darc.Action(r), expression.Expr(gsigner.Identity().String()))
	}
	gdarc = darc.NewDarc(rules, []byte{})
	dBuf, err := gdarc.ToProto()
	log.ErrFatal(err)
	ct.Store(byzcoin.NewInstanceID(gdarc.GetBaseID()), dBuf, "darc", gdarc.GetBaseID())
	return ct
}

func (ct *cvTest) Store(key byzcoin.InstanceID, value []byte, contractID string, darcID darc.ID) {
	k := string(key.Slice())
	ct.values[k] = value
	ct.contractIDs[k] = contractID
	ct.darcIDs[k] = darcID
	ct.index++
}

func (ct cvTest) setSignatureCounter(id string, v uint64) {
	key := sha256.Sum256([]byte("signercounter_" + id))
	verBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(verBuf, v)
	ct.values[string(key[:])] = verBuf
	ct.contractIDs[string(key[:])] = ""
	ct.darcIDs[string(key[:])] = darc.ID([]byte{})
}

func (ct cvTest) GetValues(key []byte) (value []byte, version uint64, contractID string, darcID darc.ID, err error) {
	return ct.values[string(key)], 0, ct.contractIDs[string(key)], ct.darcIDs[string(key)], nil
}
func (ct cvTest) GetValue(key []byte) ([]byte, error) {
	return ct.values[string(key)], nil
}
func (ct cvTest) GetContractID(key []byte) (string, error) {
	return ct.contractIDs[string(key)], nil
}
func (ct cvTest) GetProof(key []byte) (*trie.Proof, error) {
	return nil, errors.New("not implemented")
}

func (ct cvTest) GetIndex() int {
	return ct.index
}

func (account *contractAccount) createAccount(amount []byte) (inst byzcoin.InstanceID, err error) {
	// Creates a new coin account
	var ContractCoinID = "coin"

	accFactory, found := account.s.GetContractConstructor(ContractCoinID)
	if !found {
		return byzcoin.InstanceID{}, errors.New("couldn't find contract coin")
	}
	acc, err := accFactory(nil)

	accSpawnCt := newCT("spawn:coin")
	accSpawnCt.setSignatureCounter(gsigner.Identity().String(), 0)

	accSpawnInst := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gdarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractCoinID,
		},
		SignerIdentities: []darc.Identity{gsigner.Identity()},
		SignerCounter:    []uint64{1},
	}

	_, _, nosp := acc.Spawn(accSpawnCt, accSpawnInst, []byzcoin.Coin{})
	if nosp != nil {
		return byzcoin.InstanceID{}, errors.New("couldn't spawn account")
	}

	//Filling the account with money (specially for bidders)
	accCreditCt := newCT("invoke:mint")
	accCreditCt.setSignatureCounter(gsigner.Identity().String(), 0)

	accCreditInst := byzcoin.Instruction{
		InstanceID: accSpawnInst.InstanceID,
		Invoke: &byzcoin.Invoke{
			Command: "mint",
			Args:    byzcoin.Arguments{{Name: "coins", Value: amount}},
		},
		SignerIdentities: []darc.Identity{gsigner.Identity()},
		SignerCounter:    []uint64{1},
	}

	_, _, noinv := acc.Invoke(accCreditCt, accCreditInst, []byzcoin.Coin{})
	if noinv != nil {
		return byzcoin.InstanceID{}, errors.New("couldn't credit account")
	}
	return accSpawnInst.InstanceID, nil
}
