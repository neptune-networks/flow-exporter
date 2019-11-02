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
