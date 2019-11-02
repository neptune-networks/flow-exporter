package asndb

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const url string = "http://www.cidr-report.org/as2.0/asn.txt"

// Fetch ...
func Fetch() map[int]string {
	log.Info("Fetching up to date AS database")

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return parse(body)
}

func parse(responseBody []byte) map[int]string {
	asns := make(map[int]string)

	asnsByLine := strings.Split(string(responseBody), "\n")

	for _, asnLine := range asnsByLine {
		// Is this necessary? I think the match below will catch this.
		if asnLine == "" {
			continue
		}

		matchedASN := regexp.MustCompile(`([\d]+)\s+(.*),\s(\w{2})`).FindStringSubmatch(asnLine)
		if matchedASN == nil {
			continue
		}

		asn, err := strconv.Atoi(matchedASN[1])
		if err != nil {
			panic(err)
		}

		asns[asn] = strings.ToValidUTF8(matchedASN[2], "")
	}

	return asns
}
