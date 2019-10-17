package main

import (
	"encoding/json"
	"fmt"
	"github.com/shopify/sarama"
	"log"
	"os"
	"os/signal"
)

type flow struct {
	SourceAS      int    `json:"as_src"`
	DestinationAS int    `json:"as_dst"`
	SourceIP      string `json:"ip_dst"`
	DestinationIP string `json:"ip_src"`
	Bytes         int    `json:"bytes"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <broker> <topic>\n", os.Args[0])
		os.Exit(1)
	}

	broker := os.Args[1]
	topic := os.Args[2]

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

	sourceCounts := make(map[int]int)
	destinationCounts := make(map[int]int)

	consumed := 0

ConsumerLoop:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var flow flow
			json.Unmarshal([]byte(msg.Value), &flow)

			if flow.SourceAS != 0 {
				sourceCounts[flow.SourceAS] += flow.Bytes
			} else if flow.DestinationAS != 0 {
				destinationCounts[flow.DestinationAS] += flow.Bytes
			}

			fmt.Printf("Source AS: %d\nDestination AS: %d\nBytes: %d\n\n", flow.SourceAS, flow.DestinationAS, flow.Bytes)
			consumed++
		case <-signals:
			break ConsumerLoop
		}
	}

	log.Println("Source data:", sourceCounts)
	log.Println("Destination data:", sourceCounts)
}
