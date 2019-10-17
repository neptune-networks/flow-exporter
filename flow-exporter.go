package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shopify/sarama"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
)

type flow struct {
	SourceAS      int    `json:"as_src"`
	DestinationAS int    `json:"as_dst"`
	SourceIP      string `json:"ip_dst"`
	DestinationIP string `json:"ip_src"`
	Bytes         int    `json:"bytes"`
}

var (
	flowTrafficBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flow_traffic_bytes",
			Help: "Bytes received.",
		},
		[]string{"source_as", "destination_as"},
	)
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <broker> <topic>\n", os.Args[0])
		os.Exit(1)
	}

	broker := os.Args[1]
	topic := os.Args[2]

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()
	createConsumer(broker, topic)
}

func createConsumer(broker string, topic string) {
	consumer, err := sarama.NewConsumer([]string{broker}, nil)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := partitionConsumer.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	consumed := 0

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var flow flow
			json.Unmarshal([]byte(msg.Value), &flow)

			flowTrafficBytes.With(
				prometheus.Labels{
					"source_as":      strconv.Itoa(flow.SourceAS),
					"destination_as": strconv.Itoa(flow.DestinationAS),
				},
			).Add(float64(flow.Bytes))
			consumed++
			log.Printf("Consumed message offset %d\n", msg.Offset)
		case <-signals:
			break ConsumerLoop
		}
	}
}
