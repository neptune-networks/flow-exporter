# ASN Database

This package fetches an up to date list of Autonomous System Numbers, their names, and returns them in a map.

## Usage

```go
package main

import (
	"github.com/neptune-networks/flow-exporter/pkg/asndb"
	"log"
)

func main() {
	asnDB := asndb.New()
	asns, err := asnDB.Fetch()
	if err != nil {
		log.Fatalf("Error loading ASN Database: %s\n", err)
	}

	log.Printf(asns[397143])
	// NEPTUNE-NETWORKS - Neptune Networks
}
```

## Advanced Usage

By default, this package uses [CIDR Report's](http://www.cidr-report.org/as2.0/asn.txt) dump of ASN to name mappings. You can change the underlying source of ASN to name data by instantiating your own `ASNDB` struct with a different URL and corresponding regex to parse it. Below is an example on how to do this with ARIN's ASN dumps:

```go
package main

import (
	"log"
	"net/http"

	"github.com/neptune-networks/flow-exporter/pkg/asndb"
)

func main() {
	asnDB := &asndb.ASNDB{
		URL:       "http://ftp.arin.net/info/asn.txt",
		ASNFormat: `^\s([\d]+)\s+([\w-_]+)`,
		HTTP:      &http.Client{},
	}

	asns, err := asnDB.Fetch()
	if err != nil {
		log.Fatalf("Error loading ASN Database: %s\n", err)
	}

	log.Printf(asns[397143])
	// NEPTUNE-NETWORKS
}
```
