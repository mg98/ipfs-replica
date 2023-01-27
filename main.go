package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"io"
	"log"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/korovkin/limiter"
	rg "github.com/redislabs/redisgraph-go"
	"github.com/trudi-group/ipfs-metric-exporter/metricplugin"
)

// routingKeyFmt is the format of the RabbitMQ routing key that we subscribe to (taken from metricplugin source code).
const routingKeyFmt = "monitor.%s.bitswap_messages"

// monitorHost is the host of kubo-mexport instance inside (!) the ipfs-metric-exporter network.
// See: https://github.com/trudi-group/ipfs-metric-exporter/blob/master/docker-compose/docker-compose.yml.
const monitorHost = "docker_compose_monitor_01"

// dataDir is the path to the folder where IPFS data blocks should be exported to.
const dataDir = "data"

// rgHost is the host of the RedisGraph database.
var rgHost = "127.0.0.1:6379"

// rmqURL is the host of the RabbitMQ instance.
var rmqURL = "amqp://127.0.0.1:5672/%2f"

// graph is the database interface.
var graph rg.Graph

// jobs carries the asynchronous tasks of DownloadRawFile.
var jobs *limiter.ConcurrencyLimiter

// ipfsTimeout defines the timeout for CID requests to IPFS.
var ipfsTimeout time.Duration

var ctx context.Context

func init() {
	rgHostEnv := os.Getenv("RG_HOST")
	if rgHostEnv != "" {
		rgHost = rgHostEnv
	}
	rmqURLEnv := os.Getenv("RMQ_URL")
	if rmqURLEnv != "" {
		rmqURL = rmqURLEnv
	}
}

func main() {
	// init flags
	maxConcurrentDownloads := flag.Int("climit", 10, "limit of concurrent download jobs")
	ipfsTimeoutArg := flag.Int("timeout", 10, "Timeout in seconds when retrieving a block from IPFS")
	logOutput := flag.Bool("log-output", false, "If set, info/debug logs on the progress are written to a file")
	logEvents := flag.Bool("log-events", false, "If set, processing events are exported to a JSON file")
	flag.Parse()

	ipfsTimeout = time.Second * time.Duration(*ipfsTimeoutArg)

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var logF *os.File
	if *logOutput {
		var err error
		logF, err = os.OpenFile(
			fmt.Sprintf("execution_%s.log", time.Now().Format("20060201150405")),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		log.SetOutput(logF)
	}
	defer logF.Close()

	var eventsLogFile *os.File
	if *logEvents {
		var err error
		eventsLogFile, err = os.OpenFile(
			fmt.Sprintf("events_%s.json", time.Now().Format("20060201150405")),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err != nil {
			log.Fatalf("error creating event log file: %v", err)
		}
	}

	// connect to redis graph
	log.Println("Connecting to Redis... ")
	conn, err := redis.Dial("tcp", rgHost)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	graph = rg.GraphNew("ipfs", conn)

	// connect to ipfs
	log.Println("Connecting to IPFS... ")
	node, err := NewIPFSNode(ctx)
	if err != nil {
		panic(err)
	}
	fetcher := NewIPFSFetcher(ctx, node, &graph, dataDir)

	jobs = limiter.NewConcurrencyLimiter(*maxConcurrentDownloads)

	// connect to and setup rabbitmq
	log.Println("Connecting to RabbitMQ... ")
	rmq, err := amqp.Dial(rmqURL)
	if err != nil {
		log.Fatal(err)
	}
	defer rmq.Close()

	ch, err := rmq.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		metricplugin.ExchangeName, "topic", false, false, false, false, nil,
	); err != nil {
		log.Fatal(err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := ch.QueueBind(
		q.Name,
		fmt.Sprintf(routingKeyFmt, monitorHost),
		metricplugin.ExchangeName,
		false,
		nil,
	); err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Waiting for messages...")
	go processMessages(fetcher, msgs, eventsLogFile)

	var forever chan struct{}
	<-forever

	if err := jobs.WaitAndClose(); err != nil {
		log.Fatal(err)
	}
}

// processMessages processes incoming sets of Bitswap messages.
func processMessages(f *IPFSFetcher, msgs <-chan amqp.Delivery, eventsLogFile *os.File) {
	for d := range msgs {
		r, err := gzip.NewReader(bytes.NewReader(d.Body))
		if err != nil {
			log.Fatal(err)
		}
		defer r.Close()

		data, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		var events []Event
		if err := json.Unmarshal(data, &events); err != nil {
			panic(err)
		}

		if eventsLogFile != nil {
			if _, err := eventsLogFile.Write(append(data, []byte("\n")...)); err != nil {
				log.Fatalf("error logging event: %v", err)
			}
		}

		for _, ev := range events {
			for _, block := range ev.BitswapMessage.WantlistEntries {
				f.Download(block.Cid, 0, nil)
			}
		}
	}
}
