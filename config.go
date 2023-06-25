package chain_metrics

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/log"
	"sort"
	"time"
)

type DBConfig struct {
	Victoria string `yaml:"victoria"`
}

type ChainConfig struct {
	BeaconEra string `yaml:"beacon_era"`
	BeaconAPI string `yaml:"beacon_api"`
	EthRPC    string `yaml:"eth_rpc"`
	OpRPC     string `yaml:"op_rpc"`
	L1        string `yaml:"l1"`
	Type      string `yaml:"type"`
	MinTime   uint64 `yaml:"min_time"`
}

type Config struct {
	DB     DBConfig                `yaml:"db"`
	Chains map[string]*ChainConfig `yaml:"chains"`
}

type ChainType string

const (
	EthereumChain ChainType = "ethereum"
	OPStackChain  ChainType = "opstack"
)

func ParseChainType(name string) (ChainType, error) {
	x := ChainType(name)
	switch x {
	case EthereumChain, OPStackChain:
		return x, nil
	default:
		return "", fmt.Errorf("unrecognized chain type: %q", name)
	}
}

type Chain struct {
	Name string
	Type ChainType

	// TODO beacon-era
	// TODO beacon-api
	EthRPC client.RPC
	EthCl  *sources.EthClient
	OpRPC  *sources.RollupClient

	L1      *Chain
	MinTime uint64
}

type System struct {
	// TODO victoria api http client

	Chains []*Chain
}

var defaultEthClConfig = &sources.EthClientConfig{
	ReceiptsCacheSize:     200,
	TransactionsCacheSize: 200,
	HeadersCacheSize:      200,
	PayloadsCacheSize:     200,
	MaxRequestsPerBatch:   20,
	MaxConcurrentRequests: 10,
	TrustRPC:              true,
	MustBePostMerge:       false,
	RPCProviderKind:       sources.RPCKindBasic,
	MethodResetDuration:   time.Minute,
}

func NewSystem(ctx context.Context, log log.Logger, cfg *Config) (*System, error) {
	byName := make(map[string]*Chain)
	for name, chCfg := range cfg.Chains {
		typ, err := ParseChainType(chCfg.Type)
		if err != nil {
			return nil, fmt.Errorf("chain %s has unrecognized type: %w", name, err)
		}

		ch := &Chain{
			Name:    name,
			Type:    typ,
			MinTime: chCfg.MinTime,
		}
		if typ == EthereumChain || typ == OPStackChain {
			if chCfg.EthRPC == "" {
				return nil, fmt.Errorf("eth-like chain %s needs eth-rpc", name)
			}
			ethRPC, err := client.NewRPC(ctx, log, chCfg.EthRPC)
			if err != nil {
				return nil, fmt.Errorf("failed to create eth RPC: %w", err)
			}
			ch.EthRPC = ethRPC
			ethCl, err := sources.NewEthClient(ethRPC, log, nil, defaultEthClConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create eth client: %w", err)
			}
			ch.EthCl = ethCl
		}
		if typ == OPStackChain {
			if chCfg.OpRPC == "" {
				return nil, fmt.Errorf("op-stack chain %s needs op-rpc", name)
			}
			rolRPC, err := client.NewRPC(ctx, log, chCfg.OpRPC)
			if err != nil {
				return nil, fmt.Errorf("failed to create op-RPC: %w", err)
			}
			ch.OpRPC = sources.NewRollupClient(rolRPC)
		}

		byName[name] = ch
	}
	for name, chCfg := range cfg.Chains {
		if chCfg.L1 != "" {
			l1Ch, ok := byName[chCfg.L1]
			if !ok {
				return nil, fmt.Errorf("%s has unknown l1 %s", name, chCfg.L1)
			}
			byName[name].L1 = l1Ch
		}
	}
	sys := &System{Chains: make([]*Chain, 0, len(byName))}
	for _, ch := range byName {
		sys.Chains = append(sys.Chains, ch)
	}
	// sort by name to make the system creation deterministic
	sort.Slice(sys.Chains, func(i, j int) bool {
		return sys.Chains[i].Name < sys.Chains[j].Name
	})
	return sys, nil
}
