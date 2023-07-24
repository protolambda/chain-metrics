package main

import (
	"bytes"
	"context"
	"fmt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"time"
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
			var chainConfig params.ChainConfig
			if err := ch.EthRPC.CallContext(ctx.Context, &chainConfig, "eth_chainConfig"); err != nil {
				return fmt.Errorf("failed to get chain config of %s: %w", ch.Name, err)
			}
			m = OPMetrics(&chainConfig)
		case EthereumChain:
			var chainConfig params.ChainConfig
			if err := ch.EthRPC.CallContext(ctx.Context, &chainConfig, "eth_chainConfig"); err != nil {
				return fmt.Errorf("failed to get chain config of %s: %w", ch.Name, err)
			}
			m = EthMetrics(&chainConfig)
		default:
			logger.Info("unhandled chain type", "type", ch.Type)
		}
		go sys.chainMetrics(ctx.Context, logger, ch, m)
	}
	<-ctx.Done()
	return sys.Close()
}

func (sys *System) Close() error {
	// TODO get hold of each chain db, and close
	return nil
}

func (sys *System) chainMetrics(ctx context.Context, log log.Logger, ch *Chain, m AggregateMetric[*BlockWithReceipts]) {

	// TODO determine buffer size
	blocks := make(chan *types.Block, 100)

	// backfiller
	go func() {
		// TODO fetch latest
		// traverse from min to latest
		// if block exists with same hash: continue
		// if block exists, but different hash, then remove overlapping metrics series
		// if above, or if block does not exist, then get it
		// pause after completing, then repeat
	}()

	// forward
	go func() {
		blockPollTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-blockPollTicker.C:
				var bl *types.Block // get latest block
				// traverse backwards until we reach a block we've seen already
				// rate-limit the traversal
			}
		}
	}()

	// transform the blocks into processable blocks with receipts
	go func() {
		for {
			select {
			case <-blocks:
				// TODO fetch receipts
				//ch.EthCl.FetchReceipts()
				// TODO send to chain buffer (has backpressure)
				ch.Buffer <- &BlockWithReceipts{Block: nil, Receipts: nil}
			case <-ctx.Done():
				return
			}
		}
	}()

	// consumer
	go func() {
		blockTime := func(b *BlockWithReceipts) int64 {
			return int64(b.Block.Time())
		}

		flushTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-flushTicker.C:
				var out bytes.Buffer
				ExportJSONLines[*BlockWithReceipts](ctx, blockTime, m, &out, ch.Buffer)
				// TODO write output to victoria metrics

				// TODO remember blocks that have been written
				// never re-write blocks

			case <-ctx.Done():
				return
			}
		}
	}()
}
