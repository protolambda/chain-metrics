package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

func GweiFloat64(v *big.Int) float64 {
	if v.IsUint64() { // fast path, not exact but good enough
		return float64(v.Uint64()) / 1e9
	}
	fl := new(big.Float).SetInt(v)
	fl = new(big.Float).Quo(fl, big.NewFloat(1e9))
	out, _ := fl.Float64()
	return out
}

var BlockNumberMetric = Metric[*types.Header]{
	Name: "block_number",
	Fn: func(hdr *types.Header) (float64, error) {
		return float64(hdr.Number.Uint64()), nil
	},
}

var BlockHashMetric = Metric[*types.Block]{
	Name: "block_hash",
	Fn: func(bl *types.Block) (float64, error) {
		h := bl.Hash()
		// we map the first 8 bytes to a float64, so we can graph changes of the hash to find divergences visually.
		// We don't do math.Float64frombits, just a regular conversion, to keep the value within a manageable range.
		return float64(binary.LittleEndian.Uint64(h[:])), nil
	},
}

var BaseFeeMetric = Metric[*types.Header]{
	Name: "block_basefee",
	Fn: func(hdr *types.Header) (float64, error) {
		return GweiFloat64(hdr.BaseFee), nil
	},
}

var TxCountMetric = Metric[*types.Block]{
	Name: "block_tx_count",
	Fn: func(elem *types.Block) (float64, error) {
		return float64(len(elem.Transactions())), nil
	},
}

var BlockSizeMetric = Metric[*types.Block]{
	Name: "block_size",
	Fn: func(elem *types.Block) (float64, error) {
		return float64(elem.Size()), nil
	},
}

var BlockWithdrawalsMetric = Histogram[*types.Block](
	"block_withdrawals",
	[]float64{},
	func(elem *types.Block, add func(v float64)) error {
		for _, w := range elem.Withdrawals() {
			add(float64(w.Amount))
		}
		return nil
	},
)

var BlockTxTypeUsageMetric = ParametrizedMetric[*types.Block](
	"block_tx_type_usage",
	"tx_type",
	[]string{"0", "1", "2", "3", "126", "other"},
	func(elem *types.Block, dest []float64) error {
		for _, tx := range elem.Transactions() {
			switch tx.Type() {
			case 0:
				dest[0] += 1
			case 1:
				dest[1] += 1
			case 2:
				dest[2] += 1
			case 3:
				dest[3] += 1
			case types.DepositTxType:
				dest[4] += 1
			default:
				dest[5] += 1
			}
		}
		return nil
	},
)

// fee histogram bound values, in gwei
var feeBounds = []float64{
	0.001,
	0.01,
	0.1,
	1,
	10,
	100,
	1000,
	10000,
}

func TxHistogram(name string, bounds []float64,
	fn func(bl *types.Block, tx *types.Transaction) float64) AggregateMetric[*types.Block] {
	return Histogram[*types.Block](
		name,
		feeBounds,
		func(bl *types.Block, add func(v float64)) error {
			for _, tx := range bl.Transactions() {
				add(fn(bl, tx))
			}
			return nil
		})
}

type BlockWithReceipts struct {
	Block    *types.Block
	Receipts []*types.Receipt
}

func ReceiptHistogram(name string, bounds []float64,
	fn func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64) AggregateMetric[*BlockWithReceipts] {
	return Histogram[*BlockWithReceipts](
		name,
		feeBounds,
		func(blr *BlockWithReceipts, add func(v float64)) error {
			for i, tx := range blr.Block.Transactions() {
				add(fn(blr.Block, tx, blr.Receipts[i]))
			}
			return nil
		})
}

var PriorityFeeHistogram = TxHistogram("tx_priority_fee", feeBounds,
	func(bl *types.Block, tx *types.Transaction) float64 {
		return GweiFloat64(tx.EffectiveGasTipValue(bl.BaseFee()))
	})

var TxGasLimitHistogram = TxHistogram("tx_gas_limit", []float64{
	0,
	21_000,
	50_000,
	100_000,
	250_000,
	1_000_000,
	4_000_000,
	8_000_000,
	15_000_000,
	30_000_000,
},
	func(bl *types.Block, tx *types.Transaction) float64 {
		return float64(tx.Gas())
	})

var TxNonceHistogram = ReceiptHistogram("tx_nonce", []float64{0, 1, 5, 10, 100, 1000, 10_000, 100_000},
	func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64 {
		nonce := tx.EffectiveNonce()
		if nonce != nil {
			return float64(*nonce)
		}
		if rec.DepositNonce != nil {
			return float64(*nonce)
		}
		return -1
	})

var TxSizeHistogram = TxHistogram("tx_size", []float64{
	100,
	1000,
	10_000,
	20_000,
	40_000,
	128_000,
	1000_000,
}, func(bl *types.Block, tx *types.Transaction) float64 {
	return float64(tx.Size())
})

var TxGasUsageHistogram = ReceiptHistogram("tx_gas_usage", []float64{},
	func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64 {
		return float64(rec.GasUsed)
	})

var TxFeeHistogram = ReceiptHistogram("tx_fee", []float64{},
	func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64 {
		return GweiFloat64(new(big.Int).Mul(new(big.Int).SetUint64(rec.GasUsed), rec.EffectiveGasPrice))
	})

var BlockTxLogsHistogram = ReceiptHistogram("block_tx_logs", []float64{},
	func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64 {
		return float64(len(rec.Logs))
	})

var BlockTxStatus = ParametrizedMetric[*BlockWithReceipts]("block_tx_status", "status", []string{"success", "failed"},
	func(elem *BlockWithReceipts, dest []float64) error {
		for _, rec := range elem.Receipts {
			if rec.Status == types.ReceiptStatusSuccessful {
				dest[0] += 1
			} else {
				dest[1] += 1
			}
		}
		return nil
	})

var BlockDeployTxs = Metric[*types.Block]{
	Name: "block_deploy_txs",
	Fn: func(elem *types.Block) (float64, error) {
		n := 0
		for _, tx := range elem.Transactions() {
			if tx.To() == nil {
				n += 1
			}
		}
		return float64(n), nil
	},
}

var BlockTxL1CostHistogram = ReceiptHistogram("block_tx_l1_cost", []float64{},
	func(bl *types.Block, tx *types.Transaction, rec *types.Receipt) float64 {
		return GweiFloat64(rec.L1Fee)
	})

var RollupDataHistogram = func(chCfg *params.ChainConfig) AggregateMetric[*types.Block] {
	return TxHistogram("tx_rollup_data_gas", []float64{
		0, 100, 1000, 10_000, 100_000, 1_000_000, 10_000_000,
	}, func(bl *types.Block, tx *types.Transaction) float64 {
		return float64(tx.RollupDataGas().DataGas(bl.Time(), chCfg))
	})
}

func MakeCalldataStats(l1ChId uint64) AggregateMetric[*types.Block] {
	inboxes, ok := InboxesByL1[l1ChId]
	if !ok {
		panic(fmt.Errorf("unknown L1: %d", l1ChId))
	}
	addrs := make([]common.Address, len(inboxes))
	addrToIndex := make(map[common.Address]int)
	for addr := range inboxes {
		addrToIndex[addr] = len(addrs)
		addrs = append(addrs, addr)
	}
	names := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		names = append(names, inboxes[addr].Name)
	}
	names = append(names, "contract deploys", "unknown method", "other")

	return ParametrizedMetric[*types.Block]("calldata_txs", "inbox", names, func(elem *types.Block, dest []float64) error {
		for _, tx := range elem.Transactions() {
			to := tx.To()
			if to == nil {
				// count as contract-deploy
				dest[len(dest)-3] += float64(tx.Size())
				continue
			}
			inbox, ok := inboxes[*to]
			if !ok {
				// count as other
				dest[len(dest)-1] += float64(tx.Size())
				continue
			}
			found := len(inbox.MethodSig) == 0
			for _, sig := range inbox.MethodSig {
				if bytes.HasPrefix(tx.Data(), sig[:]) {
					found = true
					break
				}
			}
			if found {
				inboxIndex := addrToIndex[*to]
				dest[inboxIndex] += float64(tx.Size())
			} else {
				// count as unknown method
				dest[len(dest)-2] += float64(tx.Size())
			}
		}
		return nil
	})
}

var EthMetrics = func(chCfg *params.ChainConfig) AggregateMetric[*BlockWithReceipts] {
	return CombineAggregates[*BlockWithReceipts](
		TransformAggregate[*types.Block, *BlockWithReceipts](
			func(b *BlockWithReceipts) *types.Block {
				return b.Block
			},
			CombineAggregates[*types.Block](
				TransformAggregate[*types.Header, *types.Block](
					func(b *types.Block) *types.Header {
						return b.Header()
					},
					Aggregate[*types.Header](
						BlockNumberMetric,
						BaseFeeMetric,
					),
				),
				Aggregate[*types.Block](
					BlockHashMetric,
					TxCountMetric,
					BlockSizeMetric,
					BlockDeployTxs,
				),
				BlockWithdrawalsMetric,
				TxGasLimitHistogram,
				BlockTxTypeUsageMetric,
				PriorityFeeHistogram,
				TxSizeHistogram,
				RollupDataHistogram(chCfg),
			),
		),
		BlockTxStatus,
		TxNonceHistogram,
		TxGasUsageHistogram,
		TxFeeHistogram,
		BlockTxLogsHistogram,
		BlockTxL1CostHistogram,
	)
}
