package crown

import (
	"bytes"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

// network constants
const (
	MainnetMagic wire.BitcoinNet = 0xb8ebb3df
	TestnetMagic wire.BitcoinNet = 0x0f180e06
)

// parser parameters
var (
	MainNetParams chaincfg.Params
	TestNetParams chaincfg.Params
)

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic
	MainNetParams.PubKeyHashAddrID = []byte{0x01, 0x75, 0x07}
	MainNetParams.ScriptHashAddrID = []byte{0x01, 0x74, 0xF1}

	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Net = TestnetMagic
	TestNetParams.PubKeyHashAddrID = []byte{0x01, 0x7A, 0xCD, 0x67}
	TestNetParams.ScriptHashAddrID = []byte{0x01, 0x7A, 0xCD, 0x51}
}

// CrownParser handle
type CrownParser struct {
	*btc.BitcoinLikeParser
	baseparser *bchain.BaseParser
}

// NewCrownParser returns new CrownParser instance
func NewCrownParser(params *chaincfg.Params, c *btc.Configuration) *CrownParser {
	return &CrownParser{
		BitcoinLikeParser: btc.NewBitcoinLikeParser(params, c),
		baseparser:        &bchain.BaseParser{},
	}
}

// GetChainParams contains network parameters for the mainnet Crown network
// and the Crown Testnet network
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	case "test":
		return &TestNetParams
	default:
		return &MainNetParams
	}
}

func (p *CrownParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	return p.baseparser.PackTx(tx, height, blockTime)
}

// UnpackTx unpacks transaction from protobuf byte array
func (p *CrownParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	return p.baseparser.UnpackTx(buf)
}

func (p *CrownParser) ParseBlock(b []byte) (*bchain.Block, error) {
	w := wire.MsgBlock{}
	r := bytes.NewReader(b)

	if err := w.BtcDecode(r, 0, wire.BaseEncoding); err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: w.Header.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}
