package main

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
)

var (
	ConfigLocationFlag = &cli.PathFlag{
		Name:  "config",
		Usage: "path to config file",
		Value: "config.yaml",
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, opio.DefaultInterruptSignals...)
	go func() {
		<-interruptChannel
		cancel()
	}()
	app := cli.NewApp()
	app.Name = "onchain-metrics"
	app.Description = "export onchain metrics to victoria-metrics"
	app.Flags = []cli.Flag{
		ConfigLocationFlag,
	}
	app.Action = start
	if err := app.RunContext(ctx, os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func readConfig(configFilePath string) (*Config, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %q: %w", configFilePath, err)
	}
	defer f.Close()
	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config %q: %w", configFilePath, err)
	}
	return &cfg, nil
}

func start(ctx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		return err
	}
	logger := oplog.NewLogger(logCfg)

	config, err := readConfig(ctx.String(ConfigLocationFlag.Name))
	if err != nil {
		return err
	}

	sys, err := NewSystem(ctx.Context, logger, config)
	if err != nil {
		return fmt.Errorf("failed to init system: %w", err)
	}

	for _, ch := range sys.Chains {
		var m AggregateMetric[*BlockWithReceipts]
		switch ch.Type {
		case OPStackChain:

		case EthereumChain:
			var chainConfig params.ChainConfig
			if err := ch.EthRPC.CallContext(ctx.Context, &chainConfig, "eth_chainConfig"); err != nil {

			}
			m = EthMetrics(&chainConfig)
		default:
			logger.Info("unhandled chain type", "type", ch.Type)
		}
		// TODO spawn task
		ch.chainMetrics(ctx.Context, logger, m)
	}

	newFin, err := chEthCl.InfoByLabel(ctx, eth.Finalized)

	// TODO compare to prev finalized
	// TODO delete all unfinalized data
	// TODO by number, fill the gap
	// TODO

	chEthCl.FetchReceipts()

	// Event loop
	//
	// on new finalized block
	// delete all unfinalized data
	// export new finalized data
	//   - select lower bound by
	// export updated range of unfinalized data again
	//
	// on new unfinalized block
	// if not connecting with previous unfinalized block:
	//	 - remove all unfinalized data
	//   - set last block to finalized block
	// export data between last block and new unfinalized latest block

	// helper method:
	//   given tip of chain by block hash, and timestamp of start:
	//   - N max at a time:
	//   	- fetch blocks by hash
	//   	- fetch receipts by hash
	//   	- export metrics data (tagged as finalized or non-finalized with epoch-number)
}

func (ch *Chain) chainMetrics(ctx context.Context, log log.Logger, m AggregateMetric[*BlockWithReceipts]) error {

	return nil
}
