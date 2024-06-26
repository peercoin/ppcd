// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

// TestBlock tests the MsgBlock API.
func TestBlock(t *testing.T) {
	pver := ProtocolVersion

	// Block 1 header.
	prevHash := &blockOne.Header.PrevBlock
	merkleHash := &blockOne.Header.MerkleRoot
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	flags := blockOne.Header.Flags
	bh := NewBlockHeader(1, prevHash, merkleHash, bits, nonce, flags)

	// Ensure the command is expected value.
	wantCmd := "block"
	msg := NewMsgBlock(bh)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgBlock: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num addresses (varInt) + max allowed addresses.
	wantPayload := uint32(4000000)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure we get the same block header data back out.
	if !reflect.DeepEqual(&msg.Header, bh) {
		t.Errorf("NewMsgBlock: wrong block header - got %v, want %v",
			spew.Sdump(&msg.Header), spew.Sdump(bh))
	}

	// Ensure transactions are added properly.
	tx := blockOne.Transactions[0].Copy()
	msg.AddTransaction(tx)
	if !reflect.DeepEqual(msg.Transactions, blockOne.Transactions) {
		t.Errorf("AddTransaction: wrong transactions - got %v, want %v",
			spew.Sdump(msg.Transactions),
			spew.Sdump(blockOne.Transactions))
	}

	// Ensure transactions are properly cleared.
	msg.ClearTransactions()
	if len(msg.Transactions) != 0 {
		t.Errorf("ClearTransactions: wrong transactions - got %v, want %v",
			len(msg.Transactions), 0)
	}
}

// TestBlockTxHashes tests the ability to generate a slice of all transaction
// hashes from a block accurately.
func TestBlockTxHashes(t *testing.T) {
	// Block 1, transaction 1 hash.
	hashStr := "1bdae84eff34e15399335fc2a48c70fa6b0b9caf972e38b0e3bda106d223f668"
	wantHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
		return
	}

	wantHashes := []chainhash.Hash{*wantHash}
	hashes, err := blockOne.TxHashes()
	if err != nil {
		t.Errorf("TxHashes: %v", err)
	}
	if !reflect.DeepEqual(hashes, wantHashes) {
		t.Errorf("TxHashes: wrong transaction hashes - got %v, want %v",
			spew.Sdump(hashes), spew.Sdump(wantHashes))
	}
}

// TestBlockHash tests the ability to generate the hash of a block accurately.
func TestBlockHash(t *testing.T) {
	// Block 1 hash.
	hashStr := "be4e024af5071ba515c7510767f42ec9e40c5fba56775ff296658"
	wantHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Ensure the hash produced is expected.
	blockHash := blockOne.BlockHash()
	if !blockHash.IsEqual(wantHash) {
		t.Errorf("BlockHash: wrong hash - got %v, want %v",
			spew.Sprint(blockHash), spew.Sprint(wantHash))
	}
}

// TestBlockWire tests the MsgBlock wire encode and decode for various numbers
// of transaction inputs and outputs and protocol versions.
func TestBlockWire(t *testing.T) {
	tests := []struct {
		in     *MsgBlock       // Message to encode
		out    *MsgBlock       // Expected decoded message
		buf    []byte          // Wire encoding
		txLocs []TxLoc         // Expected transaction locations
		pver   uint32          // Protocol version for wire encoding
		enc    MessageEncoding // Message encoding format
	}{
		// Latest protocol version.
		{
			&blockOne,
			&blockOne,
			blockOneBytesWire,
			blockOneTxLocs,
			ProtocolVersion,
			BaseEncoding,
		},

		// Protocol version BIP0035Version.
		{
			&blockOne,
			&blockOne,
			blockOneBytesWire,
			blockOneTxLocs,
			BIP0035Version,
			BaseEncoding,
		},

		// Protocol version BIP0031Version.
		{
			&blockOne,
			&blockOne,
			blockOneBytesWire,
			blockOneTxLocs,
			BIP0031Version,
			BaseEncoding,
		},

		// Protocol version NetAddressTimeVersion.
		{
			&blockOne,
			&blockOne,
			blockOneBytesWire,
			blockOneTxLocs,
			NetAddressTimeVersion,
			BaseEncoding,
		},

		// Protocol version MultipleAddressVersion.
		{
			&blockOne,
			&blockOne,
			blockOneBytesWire,
			blockOneTxLocs,
			MultipleAddressVersion,
			BaseEncoding,
		},
		// TODO(roasbeef): add case for witnessy block
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from wire format.
		var msg MsgBlock
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestBlockWireErrors performs negative tests against wire encode and decode
// of MsgBlock to confirm error paths work correctly.
func TestBlockWireErrors(t *testing.T) {
	// Use protocol version 60002 specifically here instead of the latest
	// because the test data is using bytes encoded with that protocol
	// version.
	pver := uint32(60002)

	tests := []struct {
		in       *MsgBlock       // Value to encode
		buf      []byte          // Wire encoding
		pver     uint32          // Protocol version for wire encoding
		enc      MessageEncoding // Message encoding format
		max      int             // Max size of fixed buffer to induce errors
		writeErr error           // Expected write error
		readErr  error           // Expected read error
	}{
		// Force error in version.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
		// Force error in prev block hash.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 4, io.ErrShortWrite, io.EOF},
		// Force error in merkle root.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 36, io.ErrShortWrite, io.EOF},
		// Force error in timestamp.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 68, io.ErrShortWrite, io.EOF},
		// Force error in difficulty bits.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 72, io.ErrShortWrite, io.EOF},
		// Force error in header nonce.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 76, io.ErrShortWrite, io.EOF},
		// ppc: Force error in header flags.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 80, io.ErrShortWrite, io.EOF},
		// Force error in transaction count.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 84, io.ErrShortWrite, io.EOF},
		// Force error in transactions.
		{&blockOne, blockOneBytesWire, pver, BaseEncoding, 85, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, test.enc)
		if err != test.writeErr {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// Decode from wire format.
		var msg MsgBlock
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if err != test.readErr {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

// TestBlockSerialize tests MsgBlock serialize and deserialize.
func TestBlockSerialize(t *testing.T) {
	tests := []struct {
		in     *MsgBlock // Message to encode
		out    *MsgBlock // Expected decoded message
		buf    []byte    // Serialized data
		txLocs []TxLoc   // Expected transaction locations
	}{
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Serialize the block.
		var buf bytes.Buffer
		err := test.in.Serialize(&buf)
		if err != nil {
			t.Errorf("Serialize #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("Serialize #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Deserialize the block.
		var block MsgBlock
		rbuf := bytes.NewReader(test.buf)
		err = block.Deserialize(rbuf)
		if err != nil {
			t.Errorf("Deserialize #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&block, test.out) {
			t.Errorf("Deserialize #%d\n got: %s want: %s", i,
				spew.Sdump(&block), spew.Sdump(test.out))
			continue
		}

		// Deserialize the block while gathering transaction location
		// information.
		var txLocBlock MsgBlock
		br := bytes.NewBuffer(test.buf)
		txLocs, err := txLocBlock.DeserializeTxLoc(br)
		if err != nil {
			t.Errorf("DeserializeTxLoc #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&txLocBlock, test.out) {
			t.Errorf("DeserializeTxLoc #%d\n got: %s want: %s", i,
				spew.Sdump(&txLocBlock), spew.Sdump(test.out))
			continue
		}
		if !reflect.DeepEqual(txLocs, test.txLocs) {
			t.Errorf("DeserializeTxLoc #%d\n got: %s want: %s", i,
				spew.Sdump(txLocs), spew.Sdump(test.txLocs))
			continue
		}
	}
}

// TestBlockSerializeErrors performs negative tests against wire encode and
// decode of MsgBlock to confirm error paths work correctly.
func TestBlockSerializeErrors(t *testing.T) {
	tests := []struct {
		in       *MsgBlock // Value to encode
		buf      []byte    // Serialized data
		max      int       // Max size of fixed buffer to induce errors
		writeErr error     // Expected write error
		readErr  error     // Expected read error
	}{
		// Force error in version.
		{&blockOne, blockOneBytes, 0, io.ErrShortWrite, io.EOF},
		// Force error in prev block hash.
		{&blockOne, blockOneBytes, 4, io.ErrShortWrite, io.EOF},
		// Force error in merkle root.
		{&blockOne, blockOneBytes, 36, io.ErrShortWrite, io.EOF},
		// Force error in timestamp.
		{&blockOne, blockOneBytes, 68, io.ErrShortWrite, io.EOF},
		// Force error in difficulty bits.
		{&blockOne, blockOneBytes, 72, io.ErrShortWrite, io.EOF},
		// Force error in header nonce.
		{&blockOne, blockOneBytes, 76, io.ErrShortWrite, io.EOF},
		// Force error in transaction count.
		{&blockOne, blockOneBytes, 80, io.ErrShortWrite, io.EOF},
		// Force error in transactions.
		{&blockOne, blockOneBytes, 81, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Serialize the block.
		w := newFixedWriter(test.max)
		err := test.in.Serialize(w)
		if err != test.writeErr {
			t.Errorf("Serialize #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// Deserialize the block.
		var block MsgBlock
		r := newFixedReader(test.max, test.buf)
		err = block.Deserialize(r)
		if err != test.readErr {
			t.Errorf("Deserialize #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		var txLocBlock MsgBlock
		br := bytes.NewBuffer(test.buf[0:test.max])
		_, err = txLocBlock.DeserializeTxLoc(br)
		if err != test.readErr {
			t.Errorf("DeserializeTxLoc #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

// TestBlockOverflowErrors  performs tests to ensure deserializing blocks which
// are intentionally crafted to use large values for the number of transactions
// are handled properly.  This could otherwise potentially be used as an attack
// vector.
/* todo ppc wire differs from disk, DeserializeTxLoc doesn't support wire
     either re-encode it or add seperate test case
func TestBlockOverflowErrors(t *testing.T) {
	// Use protocol version 70001 specifically here instead of the latest
	// protocol version because the test data is using bytes encoded with
	// that version.
	pver := uint32(70001)

	tests := []struct {
		buf  []byte          // Wire encoding
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding format
		err  error           // Expected error
	}{
		// Block that claims to have ~uint64(0) transactions.
		{
			[]byte{
				0x01, 0x00, 0x00, 0x00, // Version 1
				0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
				0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
				0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
				0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, // PrevBlock
				0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
				0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
				0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
				0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e, // MerkleRoot
				0x61, 0xbc, 0x66, 0x49, // Timestamp
				0xff, 0xff, 0x00, 0x1d, // Bits
				0x01, 0xe3, 0x62, 0x99, // Nonce
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, // TxnCount
			}, pver, BaseEncoding, &MessageError{},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Decode from wire format.
		var msg MsgBlock
		r := bytes.NewReader(test.buf)
		err := msg.BtcDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}

		// Deserialize from wire format.
		r = bytes.NewReader(test.buf)
		err = msg.Deserialize(r)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Deserialize #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}

		// Deserialize with transaction location info from wire format.
		br := bytes.NewBuffer(test.buf)
		_, err = msg.DeserializeTxLoc(br)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("DeserializeTxLoc #%d wrong error got: %v, "+
				"want: %v", i, err, reflect.TypeOf(test.err))
			continue
		}
	}
}
*/

// TestBlockSerializeSize performs tests to ensure the serialize size for
// various blocks is accurate.
func TestBlockSerializeSize(t *testing.T) {
	// Block with no transactions.
	noTxBlock := NewMsgBlock(&blockOne.Header)

	tests := []struct {
		in   *MsgBlock // Block to encode
		size int       // Expected serialized size
	}{
		// Block with no transactions.
		// peercoin: signature adds 1 here because of empty block sig
		{noTxBlock, 82},

		// First block in the mainnet block chain.
		{&blockOne, len(blockOneBytes)},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		serializedSize := test.in.SerializeSize()
		if serializedSize != test.size {
			t.Errorf("MsgBlock.SerializeSize: #%d got: %d, want: "+
				"%d", i, serializedSize, test.size)
			continue
		}
	}
}

// blockOne is the first block in the mainnet block chain.
var blockOne = MsgBlock{
	Header: BlockHeader{
		Version: 1,
		PrevBlock: chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
			0xe3, 0x27, 0xcd, 0x80, 0xc8, 0xb1, 0x7e, 0xfd,
			0xa4, 0xea, 0x08, 0xc5, 0x87, 0x7e, 0x95, 0xd8,
			0x77, 0x46, 0x2a, 0xb6, 0x63, 0x49, 0xd5, 0x66,
			0x71, 0x67, 0xfe, 0x32, 0x00, 0x00, 0x00, 0x00,
		}),
		MerkleRoot: chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
			0x68, 0xf6, 0x23, 0xd2, 0x06, 0xa1, 0xbd, 0xe3,
			0xb0, 0x38, 0x2e, 0x97, 0xaf, 0x9c, 0x0b, 0x6b,
			0xfa, 0x70, 0x8c, 0xa4, 0xc2, 0x5f, 0x33, 0x99,
			0x53, 0xe1, 0x34, 0xff, 0x4e, 0xe8, 0xda, 0x1b,
		}),

		Timestamp: time.Unix(0x50312e24, 0), // Sun Aug 19 2012 20:19:16 GMT+0200
		Bits:      0x1c00ffff,
		Nonce:     0x722a498e, // 1915373966
		Flags: 0x00000000,
	},
	Transactions: []*MsgTx{
		{
			Version:   1,
			Timestamp: time.Unix(0x50312c74, 0),
			TxIn: []*TxIn{
				{
					PreviousOutPoint: OutPoint{
						Hash:  chainhash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x04, 0x24, 0x2e, 0x31, 0x50, 0x02, 0x1a, 0x02,
						0x06, 0x2f, 0x50, 0x32, 0x53, 0x48, 0x2f,
					},
					Sequence: 0xffffffff,
				},
			},
			TxOut: []*TxOut{
				{
					Value: 0x94ff2870,
					PkScript: []byte{
						0x21, // OP_DATA_33
						0x02, 0xe5, 0xd9, 0x73, 0x5f, 0x12, 0xcc, 0x4a,
						0xdf, 0xce, 0x70, 0x84, 0x45, 0xc9, 0xe1, 0x09,
						0xdc, 0x94, 0x5a, 0x13, 0x53, 0x77, 0xbd, 0xdf,
						0x6a, 0x6f, 0x85, 0x67, 0x57, 0xc8, 0x8e, 0xec,
						0xb3, // 33-byte signature
						0xac, // OP_CHECKSIG
					},
				},
			},
			LockTime: 0,
		},
	},
	Signature: []byte{
		0x47, 0x30, 0x45, 0x02, 0x21, 0x00, 0x91, 0xc8,
		0x60, 0xf8, 0x69, 0xad, 0xe3, 0xc1, 0x53, 0x6a,
		0x94, 0xc7, 0xf9, 0xe5, 0x1a, 0xe7, 0x2a, 0xa0,
		0x43, 0x80, 0xf6, 0xd2, 0x21, 0x98, 0x32, 0x0d,
		0x73, 0x3a, 0x5b, 0xea, 0x6a, 0xc5, 0x02, 0x20,
		0x35, 0xe4, 0xda, 0xaf, 0xf5, 0xb1, 0x1f, 0x4d,
		0xbd, 0x93, 0xa0, 0x2c, 0xc7, 0x9d, 0x20, 0x7c,
		0xbe, 0x33, 0x19, 0x9c, 0x1c, 0x05, 0xfd, 0xe9,
		0xdf, 0x88, 0x22, 0xd9, 0x2e, 0x5c, 0x20, 0xc1,
	},
}

// Block one serialized bytes.
var blockOneBytes = []byte{
	0x01, 0x00, 0x00, 0x00, // Version 1
	0xe3, 0x27, 0xcd, 0x80, 0xc8, 0xb1, 0x7e, 0xfd,
	0xa4, 0xea, 0x08, 0xc5, 0x87, 0x7e, 0x95, 0xd8,
	0x77, 0x46, 0x2a, 0xb6, 0x63, 0x49, 0xd5, 0x66,
	0x71, 0x67, 0xfe, 0x32, 0x00, 0x00, 0x00, 0x00, // PrevBlock
	0x68, 0xf6, 0x23, 0xd2, 0x06, 0xa1, 0xbd, 0xe3,
	0xb0, 0x38, 0x2e, 0x97, 0xaf, 0x9c, 0x0b, 0x6b,
	0xfa, 0x70, 0x8c, 0xa4, 0xc2, 0x5f, 0x33, 0x99,
	0x53, 0xe1, 0x34, 0xff, 0x4e, 0xe8, 0xda, 0x1b, // MerkleRoot
	0x24, 0x2e, 0x31, 0x50, // Timestamp
	0xff, 0xff, 0x00, 0x1c, // Bits
	0x8e, 0x49, 0x2a, 0x72, // Nonce
	0x01,                   // TxnCount
	0x01, 0x00, 0x00, 0x00, // Version
	0x74, 0x2c, 0x31, 0x50, // Timestamp
	0x01, // Varint for number of transaction inputs
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Previous output hash
	0xff, 0xff, 0xff, 0xff, // Prevous output index
	0x0f, // Varint for length of signature script
	0x04, 0x24, 0x2e, 0x31, 0x50, 0x02, 0x1a, 0x02,
	0x06, 0x2f, 0x50, 0x32, 0x53, 0x48, 0x2f, // Signature script (coinbase)
	0xff, 0xff, 0xff, 0xff, // Sequence
	0x01,                                           // Varint for number of transaction outputs
	0x70, 0x28, 0xff, 0x94, 0x00, 0x00, 0x00, 0x00, // Transaction amount
	0x23, // Varint for length of pk script
	0x21, // OP_DATA_33
	0x02, 0xe5, 0xd9, 0x73, 0x5f, 0x12, 0xcc, 0x4a,
	0xdf, 0xce, 0x70, 0x84, 0x45, 0xc9, 0xe1, 0x09,
	0xdc, 0x94, 0x5a, 0x13, 0x53, 0x77, 0xbd, 0xdf,
	0x6a, 0x6f, 0x85, 0x67, 0x57, 0xc8, 0x8e, 0xec,
	0xb3,                   // 33-byte uncompressed public key
	0xac,                   // OP_CHECKSIG
	0x00, 0x00, 0x00, 0x00, // Lock time
	0x48, // todo ppc this shouldn't be here
	0x47, 0x30, 0x45, 0x02, 0x21, 0x00, 0x91, 0xc8,
	0x60, 0xf8, 0x69, 0xad, 0xe3, 0xc1, 0x53, 0x6a,
	0x94, 0xc7, 0xf9, 0xe5, 0x1a, 0xe7, 0x2a, 0xa0,
	0x43, 0x80, 0xf6, 0xd2, 0x21, 0x98, 0x32, 0x0d,
	0x73, 0x3a, 0x5b, 0xea, 0x6a, 0xc5, 0x02, 0x20,
	0x35, 0xe4, 0xda, 0xaf, 0xf5, 0xb1, 0x1f, 0x4d,
	0xbd, 0x93, 0xa0, 0x2c, 0xc7, 0x9d, 0x20, 0x7c,
	0xbe, 0x33, 0x19, 0x9c, 0x1c, 0x05, 0xfd, 0xe9,
	0xdf, 0x88, 0x22, 0xd9, 0x2e, 0x5c, 0x20, 0xc1, // Block signature
}

var blockOneBytesWire = []byte{
	0x01, 0x00, 0x00, 0x00, // Version 1
	0xe3, 0x27, 0xcd, 0x80, 0xc8, 0xb1, 0x7e, 0xfd,
	0xa4, 0xea, 0x08, 0xc5, 0x87, 0x7e, 0x95, 0xd8,
	0x77, 0x46, 0x2a, 0xb6, 0x63, 0x49, 0xd5, 0x66,
	0x71, 0x67, 0xfe, 0x32, 0x00, 0x00, 0x00, 0x00, // PrevBlock
	0x68, 0xf6, 0x23, 0xd2, 0x06, 0xa1, 0xbd, 0xe3,
	0xb0, 0x38, 0x2e, 0x97, 0xaf, 0x9c, 0x0b, 0x6b,
	0xfa, 0x70, 0x8c, 0xa4, 0xc2, 0x5f, 0x33, 0x99,
	0x53, 0xe1, 0x34, 0xff, 0x4e, 0xe8, 0xda, 0x1b, // MerkleRoot
	0x24, 0x2e, 0x31, 0x50, // Timestamp
	0xff, 0xff, 0x00, 0x1c, // Bits
	0x8e, 0x49, 0x2a, 0x72, // Nonce
	0x00, 0x00, 0x00, 0x00, // Flags
	0x01,                   // TxnCount
	0x01, 0x00, 0x00, 0x00, // Version
	0x74, 0x2c, 0x31, 0x50, // Timestamp
	0x01, // Varint for number of transaction inputs
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Previous output hash
	0xff, 0xff, 0xff, 0xff, // Prevous output index
	0x0f, // Varint for length of signature script
	0x04, 0x24, 0x2e, 0x31, 0x50, 0x02, 0x1a, 0x02,
	0x06, 0x2f, 0x50, 0x32, 0x53, 0x48, 0x2f, // Signature script (coinbase)
	0xff, 0xff, 0xff, 0xff, // Sequence
	0x01,                                           // Varint for number of transaction outputs
	0x70, 0x28, 0xff, 0x94, 0x00, 0x00, 0x00, 0x00, // Transaction amount
	0x23, // Varint for length of pk script
	0x21, // OP_DATA_33
	0x02, 0xe5, 0xd9, 0x73, 0x5f, 0x12, 0xcc, 0x4a,
	0xdf, 0xce, 0x70, 0x84, 0x45, 0xc9, 0xe1, 0x09,
	0xdc, 0x94, 0x5a, 0x13, 0x53, 0x77, 0xbd, 0xdf,
	0x6a, 0x6f, 0x85, 0x67, 0x57, 0xc8, 0x8e, 0xec,
	0xb3,                   // 33-byte uncompressed public key
	0xac,                   // OP_CHECKSIG
	0x00, 0x00, 0x00, 0x00, // Lock time
	0x48, // todo ppc this shouldn't be here
	0x47, 0x30, 0x45, 0x02, 0x21, 0x00, 0x91, 0xc8,
	0x60, 0xf8, 0x69, 0xad, 0xe3, 0xc1, 0x53, 0x6a,
	0x94, 0xc7, 0xf9, 0xe5, 0x1a, 0xe7, 0x2a, 0xa0,
	0x43, 0x80, 0xf6, 0xd2, 0x21, 0x98, 0x32, 0x0d,
	0x73, 0x3a, 0x5b, 0xea, 0x6a, 0xc5, 0x02, 0x20,
	0x35, 0xe4, 0xda, 0xaf, 0xf5, 0xb1, 0x1f, 0x4d,
	0xbd, 0x93, 0xa0, 0x2c, 0xc7, 0x9d, 0x20, 0x7c,
	0xbe, 0x33, 0x19, 0x9c, 0x1c, 0x05, 0xfd, 0xe9,
	0xdf, 0x88, 0x22, 0xd9, 0x2e, 0x5c, 0x20, 0xc1, // Block signature
}

// Transaction location information for block one transactions.
var blockOneTxLocs = []TxLoc{
	{TxStart: 81, TxLen: 114},
}
