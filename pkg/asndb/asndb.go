package asndb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const url string = "http://www.cidr-report.org/as2.0/asn.txt"
const asnFormat string = `([\d]+)\s+(.*),\s(\w{2})`

// Fetch ...
func Fetch() (map[int]string, error) {
	log.Info("Fetching up to date AS database")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Error communicating with %s", url)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response from %s", url)
	}

	return parse(body), nil
}

func parse(responseBody []byte) map[int]string {
	asns := make(map[int]string)

	lines := strings.Split(string(responseBody), "\n")

	for _, line := range lines {
		match := regexp.MustCompile(asnFormat).FindStringSubmatch(line)

		// If line doesn't match the expected format of the ASN
		if match == nil {
			continue
		}

		asn, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}

		asns[asn] = strings.ToValidUTF8(match[2], "")
	}

	return asns
}
