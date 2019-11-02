package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/neptune-networks/flow-exporter/pkg/asndb"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type config struct {
	broker string
	topic  string
	asn    int
	asns   map[int]string
}

type flow struct {
	SourceAS      int    `json:"as_src"`
	DestinationAS int    `json:"as_dst"`
	SourceIP      string `json:"ip_dst"`
	DestinationIP string `json:"ip_src"`
	Bytes         int    `json:"bytes"`
	Hostname      string `json:"label"`
}

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

func main() {
	broker := flag.String("broker", "", "The Kafka broker to connect to")
	topic := flag.String("topic", "", "The Kafka topic to consume from")
	asn := flag.Int("asn", 0, "The ASN being monitored")
	flag.Parse()

	if *broker == "" || *topic == "" || *asn == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	asns := asndb.Fetch()
	runtimeOptions := config{broker: *broker, topic: *topic, asn: *asn}

	go startPrometheusServer()
	createConsumer(runtimeOptions, asns)
}

func startPrometheusServer() {
	log.Info("Starting Prometheus web server")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9590", nil)
}

func createConsumer(options config, asnsDatabase map[int]string) {
	log.Info("Starting Kafka consumer")
	consumer, err := sarama.NewConsumer([]string{options.broker}, nil)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	partitionConsumer, err := consumer.ConsumePartition(options.topic, 0, sarama.OffsetNewest)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var flow flow
			json.Unmarshal([]byte(msg.Value), &flow)

			if flow.SourceAS == options.asn {
				flowTransmitBytesTotal.With(
					prometheus.Labels{
						"source_as":           strconv.Itoa(flow.SourceAS),
						"source_as_name":      asnsDatabase[flow.SourceAS],
						"destination_as":      strconv.Itoa(flow.DestinationAS),
						"destination_as_name": asnsDatabase[flow.DestinationAS],
						"hostname":            flow.Hostname,
					},
				).Add(float64(flow.Bytes))
			} else if flow.DestinationAS == options.asn {
				flowReceiveBytesTotal.With(
					prometheus.Labels{
						"source_as":           strconv.Itoa(flow.SourceAS),
						"source_as_name":      asnsDatabase[flow.SourceAS],
						"destination_as":      strconv.Itoa(flow.DestinationAS),
						"destination_as_name": asnsDatabase[flow.DestinationAS],
						"hostname":            flow.Hostname,
					},
				).Add(float64(flow.Bytes))
			}

			log.WithFields(log.Fields{"offset": msg.Offset}).Debug("Consumed message offset")
		case <-signals:
			break ConsumerLoop
		}
	}
}