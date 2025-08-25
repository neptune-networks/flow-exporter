package consumer

import (
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
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

// Consume ...
func Consume(brokers string, topic string, partitions string, asn int, asns map[int]string) {
	log.Info("Starting Kafka consumer")
	c, err := sarama.NewConsumer(strings.Split(brokers, ","), nil)
	if err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}

	partitionList, err := getPartitions(c, topic, partitions)
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
		for message := range messages {
			logFlow(*message, asns, asn)
		}
	}()

	wg.Wait()
	close(messages)

	if err := c.Close(); err != nil {
		log.Warnf("Failed to close consumer: %s", err)
	}
}

func getPartitions(c sarama.Consumer, topic string, partitions string) ([]int32, error) {
	if partitions == "all" {
		return c.Partitions(topic)
	}

	tmp := strings.Split(partitions, ",")
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

func logFlow(message sarama.ConsumerMessage, asns map[int]string, asn int) {
	var f flow
	json.Unmarshal([]byte(message.Value), &f)

	if f.SourceAS == asn {
		flowTransmitBytesTotal.With(
			prometheus.Labels{
				"source_as":           strconv.Itoa(f.SourceAS),
				"source_as_name":      asns[f.SourceAS],
				"destination_as":      strconv.Itoa(f.DestinationAS),
				"destination_as_name": asns[f.DestinationAS],
				"hostname":            f.Hostname,
			},
		).Add(float64(f.Bytes))
	} else if f.DestinationAS == asn {
		flowReceiveBytesTotal.With(
			prometheus.Labels{
				"source_as":           strconv.Itoa(f.SourceAS),
				"source_as_name":      asns[f.SourceAS],
				"destination_as":      strconv.Itoa(f.DestinationAS),
				"destination_as_name": asns[f.DestinationAS],
				"hostname":            f.Hostname,
			},
		).Add(float64(f.Bytes))
	}

	log.WithFields(log.Fields{"offset": message.Offset}).Debug("Consumed message offset")
}
