# Chain metrics

Onchain time-series metrics service.

Supported chains:
- Ethereum
- OP Stack

PRs for alt-chains are welcome if dependencies are kept minimal,
please first open an issue to propose changes.

## Config file

Data sources and targets are defined in a YAML config file:

```yaml
db:
  # VictoriaMetrics API endpoint
  victoria:

# Chains, more can be added
# Note that some L2 chains rely on L1 chain entries
chains:
  eth_mainnet:
    beacon_era:
    beacon_api:
    eth_rpc:
    type: ethereum
    min_time:  # TODO Merge time
  eth_goerli:
    beacon_era:
    beacon_api:
    eth_rpc:
    type: ethereum
    min_time:  # TODO Merge time
  op_mainnet:
    eth_rpc:
    op_rpc:
    type: opstack
    l1: eth_mainnet
    min_time:  # TODO Bedrock upgrade time
  op_goerli:
    eth_rpc:
    op_rpc:
    type: opstack
    l1: eth_goerli
    min_time:  # TODO Bedrock upgrade time
  base_goerli:
    eth_rpc:
    op_rpc:
    type: opstack
    l1: eth_goerli
    min_time: # TODO genesis time
  # ...
```

Types:
- `ethereum`: Ethereum Beacon-chain
- `opstack`: OP-stack chain, an L2 if `l1` attribute is specified. 

A `min_time` can be specified to enforce a lower-bound range, to ignore any legacy / unavailable history.

### `beacon_era`

An [Era-store](https://nimbus.guide/era-store.html) may optionally be used to quickly read L1 chain-data,
instead of fetching the blocks through `eth_rpc` and `beacon_api`.
This is recommended when backfilling historical data.

### `beacon_api`

A Beacon-API may be used for `ethereum`

### `eth_rpc`

A JSON-RPC source is required for receipts-related chain metrics.
Some metrics rely on receipts data. `debug_getRawReceipts` should be open to read receipts efficiently.

### `op_rpc`

Used to retrieve the rollup-config of the OP chain.

## CSV backfill into VictoriaMetrics

Historical data can be generated and inserted into victoria metrics:
```
chain-metrics backfill --start-time=... --end-time=...
```
This happens in batches, through the CSV data-insertion endpoint.
https://github.com/VictoriaMetrics/VictoriaMetrics#how-to-import-csv-data

## CSV dump

```
chain-metrics csv --start-time=... --end-time=...
```

## Live update into VictoriaMetrics

```
chain-metrics live
```

### Reorg handling

Time-series data is labeled with the L1 beacon epoch:
`epoch=%d`: If a reorg is detected, the invalidated data can be deleted by selecting by `epoch`.
VictoriaMetrics does support merging/deletion of time-series, but not partial merges/deletion.

To avoid high-cardinality, finalized epochs can be merged into a single time-series.

A retention-period can be configured in VictoriaMetrics to prune old data, even though only a partial time-series.

## License

MIT, see [`LICENSE`](./LICENSE) file.
