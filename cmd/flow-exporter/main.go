package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/neptune-networks/flow-exporter/internal/consumer"
	"github.com/neptune-networks/flow-exporter/pkg/asndb"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	brokers    = flag.String("brokers", "", "A comma separated list of Kafka brokers to connect to")
	topic      = flag.String("topic", "", "The Kafka topic to consume from")
	partitions = flag.String("partitions", "all", "The partitions to consume, can be 'all' or comma-separated numbers")
	asn        = flag.Int("asn", 0, "The ASN being monitored")
)

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

	go func() {
		log.Info("Starting Prometheus web server, available at: http://localhost:9590/metrics")
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9590", nil)
	}()

	consumer.Consume(*brokers, *topic, *partitions, *asn, asns)
}
