# Flow Exporter

Flow exporter is a tool that can take flow data (Netflow, sFlow, IPFIX) from Kafka and export it to [Prometheus](https://prometheus.io). These flow records can be helpful to visualize which autonomous systems traffic is coming from and going to.

Grafana is a great tool to visualize Prometheus data, and can be used to take the flow data and visualized as so:

![preview](https://user-images.githubusercontent.com/934497/67167662-85e57080-f36a-11e9-96e2-f6f5b3b7e5d0.png)

## Usage

The exporter can be started with:

```
./flow-exporter --broker=kafka.ny1.fqdn.com:9092 --topic=pmacct.acct --asn=15169
```

- `--broker`: The Kafka broker and corresponding port to consume flows from
- `--topic`: The Kafka topic to consume flows from
- `--asn`: The autonomous system number that the flows are being monitored from

Once running, you can view the data by visiting [http://localhost:9590/metrics](http://localhost:9590/metrics).

### Kafka Schema

Flow exporter requires a Kafka topic that has events which contain the following JSON attributes:

```json
{
	"label": "bdr1.ny1.fqdn.com",
	"as_src": 15169,
	"as_dst": 6939,
	"bytes": 52,
}
```

- `label`: The hostname of the device that the flow came from
- `as_src`: The ASN that originated the flow
- `as_dst`: The ASN that the flow was destined for
- `bytes`: The number of bytes contained in the flow

## Docker

A Dockerfile is provided for convenience. It will build the source and then run the exporter. You can use the Docker command line like so:

```
docker run -p 9590:9590 neptune-networks/flow-exporter:latest --broker=kafka.ny1.fqdn.com:9092 --topic=pmacct.acct --asn=15169
```

Or if you prefer Docker Compose:

```yml
flow-exporter:
	image: neptune-networks/flow-exporter:latest
	command: --broker=kafka.ny1.fqdn.com:9092 --topic=pmacct.acct --asn=15169
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
