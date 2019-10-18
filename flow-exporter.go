package main

import (
	"encoding/json"
	"flag"
	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strconv"
)

type config struct {
	broker    string
	topic     string
	defaultAS int
}

type flow struct {
	SourceAS      int    `json:"as_src"`
	DestinationAS int    `json:"as_dst"`
	SourceIP      string `json:"ip_dst"`
	DestinationIP string `json:"ip_src"`
	Bytes         int    `json:"bytes"`
}

var (
	flowReceiveBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flow_receive_bytes_total",
			Help: "Bytes received.",
		},
		[]string{"source_as", "destination_as"},
	)

	flowTransmitBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flow_transmit_bytes_total",
			Help: "Bytes transferred.",
		},
		[]string{"source_as", "destination_as"},
	)
)

func main() {
	broker := flag.String("broker", "", "The Kafka broker to connect to")
	topic := flag.String("topic", "", "The Kafka topic to consume from")
	defaultAS := flag.Int("defaultAS", -1, "The autonomous system number to fall back to if not provided in flow")
	flag.Parse()

	options := config{broker: *broker, topic: *topic, defaultAS: *defaultAS}

	if *broker == "" || *topic == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	go func() {
		log.Info("Starting Prometheus web server")
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9590", nil)
	}()
	createConsumer(options)
}

func createConsumer(options config) {
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

			if flow.SourceAS == 0 && flow.DestinationAS != 0 {
				if options.defaultAS != -1 {
					flow.SourceAS = options.defaultAS
				}

				flowTransmitBytesTotal.With(
					prometheus.Labels{
						"source_as":      strconv.Itoa(flow.SourceAS),
						"destination_as": strconv.Itoa(flow.DestinationAS),
					},
				).Add(float64(flow.Bytes))
			} else if flow.SourceAS != 0 && flow.DestinationAS == 0 {
				if options.defaultAS != -1 {
					flow.DestinationAS = options.defaultAS
				}

				flowReceiveBytesTotal.With(
					prometheus.Labels{
						"source_as":      strconv.Itoa(flow.SourceAS),
						"destination_as": strconv.Itoa(flow.DestinationAS),
					},
				).Add(float64(flow.Bytes))
			}

			log.Debug("Consumed message offset %d\n", msg.Offset)
		case <-signals:
			break ConsumerLoop
		}
	}
}
