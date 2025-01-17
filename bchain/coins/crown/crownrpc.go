package crown

import (
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

// CrownRPC is an interface to JSON-RPC bitcoind service.
type CrownRPC struct {
	*btc.BitcoinRPC
}

type CmdRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// NewCrownRPC returns new RPC instance.
func NewCrownRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &CrownRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}
	s.ChainConfig.SupportsEstimateFee = false

	return s, nil
}

func (c *CrownRPC) InitializeMempool(addrDescForOutpoint bchain.AddrDescForOutpointFunc, onNewTxAddr bchain.OnNewTxAddrFunc, onNewTx bchain.OnNewTxFunc) error {
	return nil
}

func (b *CrownRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" && height > 0 {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}

	glog.V(1).Info("rpc: getblock ", hash)

	res := btc.ResGetBlockThin{}
	req := CmdRequest{Method: "getblock"}
	req.Params = []interface{}{hash, true}
	err = b.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}

	txs := make([]bchain.Tx, 0, len(res.Result.Txids))
	for _, txid := range res.Result.Txids {
		tx, err := b.GetTransaction(txid)
		if err != nil {
			if err == bchain.ErrTxNotFound {
				glog.Errorf("rpc: getblock: skipping transanction in block %s due error: %s", hash, err)
				continue
			}
			return nil, err
		}
		txs = append(txs, *tx)
	}
	block := &bchain.Block{
		BlockHeader: res.Result.BlockHeader,
		Txs:         txs,
	}
	return block, nil
}

func (b *CrownRPC) GetBlockHash(height uint32) (string, error) {
	glog.V(1).Info("rpc: getblockhash ", height)

	res := btc.ResGetBlockHash{}
	req := CmdRequest{Method: "getblockhash"}
	req.Params = []interface{}{height}
	err := b.Call(&req, &res)

	if err != nil {
		return "", errors.Annotatef(err, "height %v", height)
	}
	if res.Error != nil {
		if btc.IsErrBlockNotFound(res.Error) {
			return "", bchain.ErrBlockNotFound
		}
		return "", errors.Annotatef(res.Error, "height %v", height)
	}
	return res.Result, nil
}

func (b *CrownRPC) GetBlockHeader(hash string) (*bchain.BlockHeader, error) {
	glog.V(1).Info("rpc: getblockheader")

	res := btc.ResGetBlockHeader{}
	req := CmdRequest{Method: "getblockheader"}
	req.Params = []interface{}{hash, true}
	err := b.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		if btc.IsErrBlockNotFound(res.Error) {
			return nil, bchain.ErrBlockNotFound
		}
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}
	return &res.Result, nil
}

// GetBlockBytes returns block with given hash as bytes
func (b *CrownRPC) GetBlockBytes(hash string) ([]byte, error) {
	block, err := b.GetBlockRaw(hash)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(block)
}

func (b *CrownRPC) GetBlockWithoutHeader(hash string, height uint32) (*bchain.Block, error) {
	data, err := b.GetBlockBytes(hash)
	if err != nil {
		return nil, err
	}
	block, err := b.Parser.ParseBlock(data)
	if err != nil {
		return nil, errors.Annotatef(err, "%v %v", height, hash)
	}
	block.BlockHeader.Hash = hash
	block.BlockHeader.Height = height
	return block, nil
}

func (b *CrownRPC) GetBlockInfo(hash string) (*bchain.BlockInfo, error) {
	glog.V(1).Info("rpc: getblock", hash)

	res := btc.ResGetBlockInfo{}
	req := CmdRequest{Method: "getblock"}
	req.Params = []interface{}{hash, true}
	err := b.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		if btc.IsErrBlockNotFound(res.Error) {
			return nil, bchain.ErrBlockNotFound
		}
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}
	return &res.Result, nil
}

func (b *CrownRPC) GetBlockRaw(hash string) (string, error) {
	glog.V(1).Info("rpc: getblock", hash)

	res := btc.ResGetBlockRaw{}
	req := CmdRequest{Method: "getblock"}
	req.Params = []interface{}{hash, false}
	err := b.Call(&req, &res)

	if err != nil {
		return "", errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		if btc.IsErrBlockNotFound(res.Error) {
			return "", bchain.ErrBlockNotFound
		}
		return "", errors.Annotatef(res.Error, "hash %v", hash)
	}
	return res.Result, nil
}

func (b *CrownRPC) GetBlockFull(hash string) (*bchain.Block, error) {
	glog.V(1).Info("rpc: getblock (verbosity=2) ", hash)

	res := btc.ResGetBlockFull{}
	req := CmdRequest{Method: "getblock"}
	req.Params = []interface{}{hash, 2}
	err := b.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		if btc.IsErrBlockNotFound(res.Error) {
			return nil, bchain.ErrBlockNotFound
		}
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}

	for i := range res.Result.Txs {
		tx := &res.Result.Txs[i]
		for j := range tx.Vout {
			vout := &tx.Vout[j]
			// convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
			vout.ValueSat, err = b.Parser.AmountToBigInt(vout.JsonValue)
			if err != nil {
				return nil, err
			}
			vout.JsonValue = ""
		}
	}

	return &res.Result, nil
}

func (b *CrownRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}

func (b *CrownRPC) GetTransaction(txid string) (*bchain.Tx, error) {
	r, err := b.getRawTransaction(txid)
	if err != nil {
		return nil, err
	}
	tx, err := b.Parser.ParseTxFromJson(r)
	tx.CoinSpecificData = r
	if err != nil {
		return nil, errors.Annotatef(err, "txid %v", txid)
	}
	return tx, nil
}

// GetTransactionSpecific returns json as returned by backend, with all coin specific data
func (b *CrownRPC) GetTransactionSpecific(tx *bchain.Tx) (json.RawMessage, error) {
	if csd, ok := tx.CoinSpecificData.(json.RawMessage); ok {
		return csd, nil
	}
	return b.getRawTransaction(tx.Txid)
}

func (b *CrownRPC) getRawTransaction(txid string) (json.RawMessage, error) {
	glog.V(1).Info("rpc: getrawtransaction ", txid)

	res := btc.ResGetRawTransaction{}
	req := CmdRequest{Method: "getrawtransaction"}
	req.Params = []interface{}{txid, 1}
	err := b.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "txid %v", txid)
	}
	if res.Error != nil {
		if btc.IsMissingTx(res.Error) {
			return nil, bchain.ErrTxNotFound
		}
		return nil, errors.Annotatef(res.Error, "txid %v", txid)
	}
	return res.Result, nil
}

func (b *CrownRPC) EstimateFee(blocks int) (big.Int, error) {
	// use EstimateSmartFee if EstimateFee is not supported
	if !b.ChainConfig.SupportsEstimateFee && b.ChainConfig.SupportsEstimateSmartFee {
		return b.EstimateSmartFee(blocks, true)
	}

	glog.V(1).Info("rpc: estimatefee ", blocks)

	res := btc.ResEstimateFee{}
	req := CmdRequest{Method: "estimatefee"}
	req.Params = []interface{}{blocks}
	err := b.Call(&req, &res)

	var r big.Int
	if err != nil {
		return r, err
	}
	if res.Error != nil {
		return r, res.Error
	}
	r, err = b.Parser.AmountToBigInt(res.Result)
	if err != nil {
		return r, err
	}
	return r, nil
}

// Initialize initializes RPC instance.
func (b *CrownRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewCrownParser(params, b.ChainConfig)

	b.Testnet = false
	b.Network = "livenet"

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

func (b *CrownRPC) GetMempoolEntry(txid string) (*bchain.MempoolEntry, error) {
	return nil, errors.New("GetMempoolEntry: not implemented")
}
