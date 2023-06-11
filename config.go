package chain_metrics

import (
	"fmt"
	"sort"
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

type BeaconEra struct {
	// TODO era-store
}

type BeaconAPI struct {
	// TODO eth2 api client
}

type EthRPC struct {
	// TODO ethclient
}

type OpRPC struct {
	// TODO rollup client
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
	// TODO APIs
	L1      *Chain
	MinTime uint64
}

type System struct {
	// TODO victoria api http client

	Chains []*Chain
}

func NewSystem(cfg *Config) (*System, error) {
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
