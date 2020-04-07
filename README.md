# Flow Exporter

Flow exporter is a tool that can take flow data (Netflow, sFlow, IPFIX) from Kafka and export it to [Prometheus](https://prometheus.io). These flow records can be helpful to visualize which autonomous systems traffic is coming from and going to.

Grafana is a great tool to visualize Prometheus data, and can be used to take the flow data and visualized as so:

![preview](https://user-images.githubusercontent.com/934497/67167662-85e57080-f36a-11e9-96e2-f6f5b3b7e5d0.png)

An in depth guide on setting this up on a Linux-based router can be found [here](https://brooks.sh/2019/11/17/network-flow-analysis-with-prometheus/).

## Usage

The exporter can be started with:

```
./flow-exporter --brokers=kafka.fqdn.com:9092 --topic=pmacct.acct --asn=15169
```

- `--brokers`: A comma separated list of Kafka brokers (with their corresponding ports) to consume flows from
- `--topic`: The Kafka topic to consume flows from
- `--asn`: The autonomous system number that the flows are being monitored from

Once running, you can view the data by visiting [http://localhost:9590/metrics](http://localhost:9590/metrics).

An example of the Prometheus metrics you can find are:

```
# HELP flow_receive_bytes_total Bytes received.
# TYPE flow_receive_bytes_total counter
flow_receive_bytes_total{destination_as="397143",destination_as_name="NEPTUNE-NETWORKS - Neptune Networks",hostname="border.neptunenetworks.org",source_as="10318",source_as_name="CABLEVISION S.A."} 663

# HELP flow_transmit_bytes_total Bytes transferred.
# TYPE flow_transmit_bytes_total counter
flow_transmit_bytes_total{destination_as="10318",destination_as_name="CABLEVISION S.A.",hostname="border.neptunenetworks.org",source_as="397143",source_as_name="NEPTUNE-NETWORKS - Neptune Networks"} 1137
```

Flow Exporter automatically finds the name of the ASN and adds it to the metric.

### Kafka Schema

Flow exporter requires a Kafka topic that has events which contain the following JSON attributes:

```json
{
  "label": "bdr1.fqdn.com",
  "as_src": 15169,
  "as_dst": 6939,
  "bytes": 52,
}
```

- `label`: The hostname of the device that the flow came from
- `as_src`: The ASN that originated the flow
- `as_dst`: The ASN that the flow was destined for
- `bytes`: The number of bytes contained in the flow

## [pmacct](https://github.com/pmacct/pmacct) Integration

Flow Exporter works well with pmacct, a series of tools for monitoring flows in Linux. The following pmacctd configuration can be used to collect flows on Linux, enrich them with BGP ASN data, and publish them to Kafka:

#### `/etc/pmacct/pmacctd.conf`

```
!
! pmacctd configuration example
!
! Did you know CONFIG-KEYS contains the detailed list of all configuration keys
! supported by 'nfacctd' and 'pmacctd' ?
!
! debug: true
daemonize: false
pcap_interfaces_map: /etc/pmacct/interfaces.map
pmacctd_as: longest
pmacctd_net: longest
sampling_rate: 1
!
bgp_daemon: true
bgp_daemon_ip: 127.0.0.2
bgp_daemon_port: 180
bgp_daemon_max_peers: 10
bgp_agent_map: /etc/pmacct/peering_agent.map
networks_file: /etc/pmacct/networks.lst
networks_file_no_lpm: true
!
aggregate: src_host, dst_host, src_port, dst_port, src_as, dst_as, label
!
plugins: kafka
kafka_output: json
kafka_broker_host: kafka.fqdn.com
kafka_topic: pmacct.acct
kafka_refresh_time: 5
kafka_history: 5m
kafka_history_roundoff: m
```

And the associated configurations referenced in that file:

#### `/etc/pmacct/interfaces.map`

```
ifindex=100 ifname=<INTERFACE>
```

#### `/etc/pmacct/peering_agent.map`

```
bgp_ip=<BGP_ROUTER_ID>     ip=0.0.0.0/0
```

More information on configuring pmacct can be found [here](https://github.com/pmacct/pmacct/blob/master/CONFIG-KEYS).

## Docker

A Dockerfile is provided for convenience. It will build the source and then run the exporter. You can use the Docker command line like so:

```
docker run -p 9590:9590 bswinnerton/flow-exporter:latest --brokers=kafka.fqdn.com:9092 --topic=pmacct.acct --asn=15169
```

Or if you prefer Docker Compose:

```yml
flow-exporter:
  image: bswinnerton/flow-exporter:latest
  command: --brokers=kafka.fqdn.com:9092 --topic=pmacct.acct --asn=15169
  expose:
    - 9590
```

Ideally in the same `docker-compose.yml` file as your Prometheus server to make communication easy.

## Building

The application can be compiled by running:

```
git clone https://github.com/neptune-networks/flow-exporter
cd flow-exporter
go build
```

## Releasing

To release a new version, the following commands must be run:

```
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
goreleaser --rm-dist
```
