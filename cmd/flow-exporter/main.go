package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/neptune-networks/flow-exporter/pkg/asndb"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	brokers    = flag.String("brokers", "", "A comma separated list of Kafka brokers to connect to")
	topic      = flag.String("topic", "", "The Kafka topic to consume from")
	partitions = flag.String("partitions", "all", "The partitions to consume, can be 'all' or comma-separated numbers")
	asn        = flag.Int("asn", 0, "The ASN being monitored")
)

var (
	flowReceiveBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flow_receive_bytes_total",
			Help: "Bytes received.",
		},
		[]string{"source_as", "source_as_name", "destination_as", "destination_as_name", "hostname"},
	)

	flowTransmitBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flow_transmit_bytes_total",
			Help: "Bytes transferred.",
		},
		[]string{"source_as", "source_as_name", "destination_as", "destination_as_name", "hostname"},
	)
)

type flow struct {
	SourceAS      int    `json:"as_src"`
	DestinationAS int    `json:"as_dst"`
	SourceIP      string `json:"ip_dst"`
	DestinationIP string `json:"ip_src"`
	Bytes         int    `json:"bytes"`
	Hostname      string `json:"label"`
}

func main() {
	flag.Parse()

	if *brokers == "" || *topic == "" || *asn == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Info("Fetching up to date AS database")
	asnDB := asndb.New()
	asns, err := asnDB.Fetch()
	if err != nil {
		log.Warn(err)
	}

	go startPrometheusServer()
	consume(*brokers, *topic, *asn, asns)
}

func startPrometheusServer() {
	log.Info("Starting Prometheus web server")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9590", nil)
}

func consume(brokers string, topic string, asn int, asns map[int]string) {
	c, err := sarama.NewConsumer(strings.Split(brokers, ","), nil)
	if err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}

	partitionList, err := getPartitions(c)
	if err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}

	var (
		messages = make(chan *sarama.ConsumerMessage, 256)
		closing  = make(chan struct{})
		wg       sync.WaitGroup
	)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Kill, os.Interrupt)
		<-signals
		log.Info("Shutting down consumer")
		close(closing)
	}()

	for _, partition := range partitionList {
		partitionConsumer, err := c.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			log.Fatalf("Failed to start consumer for partition %d: %s", partition, err)
		}

		go func(pc sarama.PartitionConsumer) {
			<-closing
			pc.AsyncClose()
		}(partitionConsumer)

		wg.Add(1)
		go func(partitionConsumer sarama.PartitionConsumer) {
			defer wg.Done()
			for message := range partitionConsumer.Messages() {
				messages <- message
			}
		}(partitionConsumer)
	}

	go func() {
		for msg := range messages {
			var flow flow
			json.Unmarshal([]byte(msg.Value), &flow)

			if flow.SourceAS == asn {
				flowTransmitBytesTotal.With(
					prometheus.Labels{
						"source_as":           strconv.Itoa(flow.SourceAS),
						"source_as_name":      asns[flow.SourceAS],
						"destination_as":      strconv.Itoa(flow.DestinationAS),
						"destination_as_name": asns[flow.DestinationAS],
						"hostname":            flow.Hostname,
					},
				).Add(float64(flow.Bytes))
			} else if flow.DestinationAS == asn {
				flowReceiveBytesTotal.With(
					prometheus.Labels{
						"source_as":           strconv.Itoa(flow.SourceAS),
						"source_as_name":      asns[flow.SourceAS],
						"destination_as":      strconv.Itoa(flow.DestinationAS),
						"destination_as_name": asns[flow.DestinationAS],
						"hostname":            flow.Hostname,
					},
				).Add(float64(flow.Bytes))
			}

			log.WithFields(log.Fields{"offset": msg.Offset}).Debug("Consumed message offset")
		}
	}()

	wg.Wait()
	close(messages)

	if err := c.Close(); err != nil {
		log.Warnf("Failed to close consumer: %s", err)
	}
}

func getPartitions(c sarama.Consumer) ([]int32, error) {
	if *partitions == "all" {
		return c.Partitions(*topic)
	}

	tmp := strings.Split(*partitions, ",")
	var partitionList []int32
	for i := range tmp {
		val, err := strconv.ParseInt(tmp[i], 10, 32)
		if err != nil {
			return nil, err
		}

		partitionList = append(partitionList, int32(val))
	}

	return partitionList, nil
}
