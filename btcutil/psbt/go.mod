module github.com/btcsuite/btcd/btcutil/psbt

go 1.21.4

require (
	github.com/btcsuite/btcd v0.23.4
	github.com/btcsuite/btcd/btcec/v2 v2.1.3
	github.com/btcsuite/btcd/btcutil v1.1.0
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1
)

require (
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/btcsuite/btcd/btcutil => ../

replace github.com/btcsuite/btcd => ../..
