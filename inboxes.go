package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

type Inbox struct {
	Name string
	// optional, some inboxes just process raw data (no method sig), some use multiple functions
	MethodSig [][4]byte
}

func inbox(name string, methodsig ...string) Inbox {
	out := Inbox{Name: name}
	for _, msig := range methodsig {
		if strings.HasPrefix(msig, "0x") {
			methodsig = methodsig[2:]
		}
		sig, err := hex.DecodeString(msig)
		if err != nil {
			panic(err)
		}
		if len(sig) != 4 {
			panic(fmt.Errorf("bad method sig len: %d", len(sig)))
		}
		var x [4]byte
		copy(x[:], sig)
		out.MethodSig = append(out.MethodSig, x)
	}
	return out
}

var EthMainnetRollupInboxes = map[common.Address]Inbox{
	chaincfg.Mainnet.BatchInboxAddress:                                inbox("mainnet op"),
	common.HexToAddress("0x1c479675ad559dc151f6ec7ed3fbf8cee79582b6"): inbox("mainnet arb one sequencer inbox", "0x8f111f3c"),
	common.HexToAddress("0x1c479675ad559dc151f6ec7ed3fbf8cee79582b6"): inbox("mainnet arb nova sequencer inbox", "0x8f111f3c"),
	common.HexToAddress("0x3dB52cE065f728011Ac6732222270b3F2360d919"): inbox("mainnet zksync era", "0x0c4dd810", "0x7739cbe7"),    // commitBlocks, proveBlocks
	common.HexToAddress("0xaBEA9132b05A70803a4E85094fD0e1800777fBEF"): inbox("mainnet zksync lite", "0x45269298"),                 // commitBlocks
	common.HexToAddress("0x5132A183E9F3CB7C848b0AAC5Ae0c4f0491B7aB2"): inbox("mainnet polygon zkevm", "0x5e9145c9", "0xa50a164b"), // sequenceBatches, verifyBatchesTrustedAggregator
	common.HexToAddress("0x6F54Ca6F6EdE96662024Ffd61BFd18f3f4e34DFf"): inbox("mainnet zora"),
}

var EthGoerliRollupInboxes = map[common.Address]Inbox{
	chaincfg.Goerli.BatchInboxAddress:                                 inbox("goerli op"),
	common.HexToAddress("0x8453100000000000000000000000000000000000"): inbox("goerli base"),
	common.HexToAddress("0xa997cfD539E703921fD1e3Cf25b4c241a27a4c7A"): inbox("goerli polygon zkevm", "0x5e9145c9", "0xa50a164b"),            // sequenceBatches, verifyBatchesTrustedAggregator
	common.HexToAddress("0xB949b4E3945628650862a29Abef3291F2eD52471"): inbox("goerli zksync era", "0x7739cbe7", "0x0c4dd810", "0xce9dcf16"), // proveBlocks, commitBlocks, executeBlocks
	common.HexToAddress("0x3C584eC7f0f2764CC715ac3180Ae9828465E9833"): inbox("goerli scroll alpha", "0xcb905499"),                           // ?
	common.HexToAddress("0x0484A87B144745A2E5b7c359552119B6EA2917A9"): inbox("goerli arb sequencer inbox", "0x8f111f3c"),                    // addSequencerL2BatchFromOrigin
	common.HexToAddress("0xFf00000000000000000000000000000000000421"): inbox("goerli op nightly"),
	common.HexToAddress("0xff00000000000000000000000000000000000888"): inbox("goerli op chaos"),
	common.HexToAddress("0x70BaD09280FD342D02fe64119779BC1f0791BAC2"): inbox("goerli linea", "0x4165d6dd"),
	common.HexToAddress("0xFf00000000000000000000000000000000042069"): inbox("goerli op unknown"),
	common.HexToAddress("0xff00000000000000000000000000000000000997"): inbox("goerli op internal"),
	common.HexToAddress("0x427c9a666d3b27873111cE3894712Bf64C6343A0"): inbox("goerli zora"),
}

var InboxesByL1 = map[uint64]map[common.Address]Inbox{
	1: EthMainnetRollupInboxes,
	5: EthGoerliRollupInboxes,
}
