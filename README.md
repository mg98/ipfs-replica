# IPFS Replicate

[![Test](https://github.com/mg98/ipfs-replicate/actions/workflows/test.yml/badge.svg)](https://github.com/mg98/ipfs-replicate/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/mg98/ipfs-replicate/branch/main/graph/badge.svg?token=R3OYXX1HC7)](https://codecov.io/gh/mg98/ipfs-replicate)
[![Go Report Card](https://goreportcard.com/badge/github.com/mg98/ipfs-replicate?)](https://goreportcard.com/report/github.com/mg98/ipfs-replicate)
![License](https://img.shields.io/github/license/mg98/ipfs-replicate)

This software lets you replicate the distributed DAG of content blocks in IPFS locally, based on network traces.
To this end, it replicates the data structure in a RedisGraph database and downloads raw data blocks to disk.

## How Does It Work?

This program uses [IPFS Metric Exporter](https://github.com/trudi-group/ipfs-metric-exporter),
which is a plugin to IPFS that exports (among other things) the CID requests from the P2P gossip to a RabbitMQ exchange instance.

**IPFS Replicate** subscribes to this exchange and processes incoming messages
by recursively fetching the contents of the requested CIDs
and populating the local database and data folder.

The raw data blocks are written as files to disk while the data structure is persisted in a RedisGraph database.

Furthermore, this tool allows you to export those user events.
This can be useful in combination with the locally produced data structure for analyses that also contemplate user behavior.

## Setup and Run

To quickly spin something up, you can launch the infrastructure using:

```sh
docker-compose up
```

If you want to run this program without Docker, 
you can download the [binary](https://github.com/mg98/ipfs-replicate/releases) directly
or build this project from source (`go build .`).

Note that this program depends on other components, which you can be comprehended from the [`docker-compose.yml`](./docker-compose.yml).
You might then want to adjust some [environment variables](./.env.example).

## Author Notes

This software has its origin in my [master thesis](https://marcelgregoriadis.com/master-thesis.pdf), 
where I used it to understand the type of files traded on IPFS
and to follow the sequence of file retrievals for individual peers.
This allowed me to analyze the effectiveness of alternative chunking algorithms on data deduplication.
I hope that by publishing this part of the software I can support the development of future scientific projects (or any other).

Please note that there is still a lot of room for improvement with this software.
As the biggest issue I regard the poor efficiency or _event throughput_.
Due to the nature of IPFS and the reliance on network retrievals (sometimes for heavy CID trees),
this program will not keep up with the pace of incoming Bitswap events... at all!
Although this project already leverages parallelization techniques, I think those can be further extended or optimized.

That said, contributions will be considered and are very welcome ❤️ 
