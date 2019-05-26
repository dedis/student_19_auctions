module github.com/dedis/student_19_auctions

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/stretchr/testify v1.3.0
	go.dedis.ch/cothority/v3 v3.0.2
	go.dedis.ch/kyber/v3 v3.0.2
	go.dedis.ch/onet/v3 v3.0.5
	go.dedis.ch/protobuf v1.0.6
	golang.org/x/crypto v0.0.0-20190320223903-b7391e95e576 // indirect
	golang.org/x/sys v0.0.0-20190322080309-f49334f85ddc // indirect
)

replace go.dedis.ch/cothority/v3 => ../dynasent/conode/cothority

replace go.dedis.ch/onet/v3 v3.0.5 => go.dedis.ch/onet/v3 v3.0.11
