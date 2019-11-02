package asndb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const url string = "http://www.cidr-report.org/as2.0/asn.txt"
const asnFormat string = `([\d]+)\s+(.*),\s(\w{2})`

// Fetch ...
func Fetch() (map[int]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Error communicating with %s", url)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response from %s", url)
	}

	asns, err := parse(body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response from %s", url)
	}

	return asns, nil
}

func parse(responseBody []byte) (map[int]string, error) {
	asns := make(map[int]string)

	lines := strings.Split(string(responseBody), "\n")

	for _, line := range lines {
		match := regexp.MustCompile(asnFormat).FindStringSubmatch(line)

		// If line doesn't match the expected format of the ASN, move on to the next line
		if match == nil {
			continue
		}

		asn, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("Error converting string to integer: %s", match[1])
		}

		asns[asn] = strings.ToValidUTF8(match[2], "")
	}

	return asns, nil
}
