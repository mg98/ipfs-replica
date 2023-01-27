# IPFS Replicate

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/mg98/ipfs-replicate)
[![Test](https://github.com/mg98/ipfs-replicate/actions/workflows/test.yml/badge.svg)](https://github.com/mg98/ipfs-replicate/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/mg98/ipfs-replicate/branch/main/graph/badge.svg?token=R3OYXX1HC7)](https://codecov.io/gh/mg98/ipfs-replicate)
[![Go Report Card](https://goreportcard.com/badge/github.com/mg98/ipfs-replicate?)](https://goreportcard.com/report/github.com/mg98/ipfs-replicate)
![License](https://img.shields.io/github/license/mg98/ipfs-replicate)

This software lets you replicate the distributed DAG of content blocks in IPFS locally, based on network traces.
To this end, it reproduces the structure in a RedisGraph database and downloads raw data blocks to disk.

## How Does It Work?

This program is meant to be used in combination with the [IPFS Metric Exporter](https://github.com/trudi-group/ipfs-metric-exporter),
which is a plugin to IPFS that exports (among other things) the CID requests from the P2P gossip to a RabbitMQ exchange instance.

**IPFS Replicate** subscribes to this exchange and processes incoming messages 
by recursively fetching the contents of the requested CIDs
and populating the local database and data folder.

Furthermore, this tool allows you to export those user events.
This can be useful in combination with the locally produced data structure for analyses that also contemplates user behavior.

## Setup and Run

As mentioned in the previous section, this program depends on IPFS traces produced by another project.
Learn how to build and run this project [here](https://github.com/trudi-group/ipfs-metric-exporter#building).

It has to be running **before and while** the execution of _this_ software!

Furthermore, this software requires a running RedisGraph instance.
To quickly spin something up, you can use the following command.

```sh
docker run -p 6379:6379 redislabs/redisgraph
```

Finally, you can run this program by executing the [binary](https://github.com/mg98/ipfs-replicate/releases).

If needed, you can adjust some [environment variables](./.env.example). 

Also, take note of the CLI options (`./ipfs-replica --help`).

To build this project from source, you can also clone this repository and build using Go (`go build .`).

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