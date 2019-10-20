package main

import (
	"encoding/json"
	"flag"
	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
)

type config struct {
	broker    string
	topic     string
	defaultAS int
	asns      map[int]string
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
	defaultAS := flag.Int("defaultAS", -1, "The autonomous system number to fall back to if not provided in flow")
	flag.Parse()

	asns := fetchASDatabase()
	runtimeOptions := config{broker: *broker, topic: *topic, defaultAS: *defaultAS}

	if *broker == "" || *topic == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	go startPrometheusServer()
	createConsumer(runtimeOptions, asns)
}

func fetchASDatabase() map[int]string {
	log.Info("Fetching up to date AS database")
	resp, err := http.Get("http://www.cidr-report.org/as2.0/asn.txt")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	asns := make(map[int]string)

	rawASNs := strings.Split(string(body), "\n")

	for _, rawASN := range rawASNs {
		if rawASN == "" {
			continue
		}

		parsedASN := regexp.MustCompile(`([\d]+)\s+(.*),\s(\w{2})`).FindStringSubmatch(rawASN)
		if parsedASN == nil {
			continue
		}

		asn, err := strconv.Atoi(parsedASN[1])
		if err != nil {
			panic(err)
		}

		asns[asn] = strings.ToValidUTF8(parsedASN[2], "")
	}

	return asns
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

			if flow.SourceAS == 0 && flow.DestinationAS != 0 {
				if options.defaultAS != -1 {
					flow.SourceAS = options.defaultAS
				}

				flowTransmitBytesTotal.With(
					prometheus.Labels{
						"source_as":           strconv.Itoa(flow.SourceAS),
						"source_as_name":      asnsDatabase[flow.SourceAS],
						"destination_as":      strconv.Itoa(flow.DestinationAS),
						"destination_as_name": asnsDatabase[flow.DestinationAS],
						"hostname":            flow.Hostname,
					},
				).Add(float64(flow.Bytes))
			} else if flow.SourceAS != 0 && flow.DestinationAS == 0 {
				if options.defaultAS != -1 {
					flow.DestinationAS = options.defaultAS
				}

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
