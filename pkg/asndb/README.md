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

You can change the underlying source being used to query the ASNs by instantiating your own `ASNDB` struct and regex to parse it:

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
