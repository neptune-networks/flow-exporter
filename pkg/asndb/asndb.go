package asndb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type httpClient interface {
	Get(string) (*http.Response, error)
}

// ASNDB fetches a list of Autonomous System Numbers and returns them as a map
type ASNDB struct {
	url       string
	asnFormat string
	http      httpClient
}

// New returns a new instance of the ASNDB
func New() *ASNDB {
	return &ASNDB{
		url:       "http://www.cidr-report.org/as2.0/asn.txt",
		asnFormat: `([\d]+)\s+(.*),\s(\w{2})`,
		http:      &http.Client{},
	}
}

// Fetch fetches and parses the ASN data from a remote site into a map
func (db ASNDB) Fetch() (map[int]string, error) {
	resp, err := db.fetch()
	if err != nil {
		return nil, err
	}

	asns, err := db.parse(resp)
	if err != nil {
		return nil, err
	}

	return asns, nil
}

func (db ASNDB) fetch() ([]byte, error) {
	resp, err := db.http.Get(db.url)
	if err != nil {
		return nil, fmt.Errorf("Error communicating with %s", db.url)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return nil, fmt.Errorf("Error reading response from %s", db.url)
	}

	return body, nil
}

func (db ASNDB) parse(responseBody []byte) (map[int]string, error) {
	asns := make(map[int]string)

	lines := strings.Split(string(responseBody), "\n")

	for _, line := range lines {
		match := regexp.MustCompile(db.asnFormat).FindStringSubmatch(line)

		if match != nil {
			asn, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, fmt.Errorf("Error converting string to integer: %s", match[1])
			}

			asns[asn] = strings.ToValidUTF8(match[2], "")
		}
	}

	return asns, nil
}
