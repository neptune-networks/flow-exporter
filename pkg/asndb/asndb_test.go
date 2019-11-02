package asndb

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchWithReachableServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	}))

	defer server.Close()

	_, err := fetch(server.URL)
	if err != nil {
		t.Errorf("Expected to receive response from server, instead received: %s", err.Error())
	}
}

func TestFetchWithUnreachableServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	defer server.Close()

	_, err := fetch(server.URL)
	if err != nil {
		t.Errorf("Expected to not receive response from server, but did: %s", err.Error())
	}
}

func TestParseWithValidResponse(t *testing.T) {
	expected := make(map[int]string)
	expected[0] = "-Reserved AS-"
	expected[1] = "LVLT-1 - Level 3 Parent, LLC"
	expected[2] = "UDEL-DCN - University of Delaware"
	expected[3] = "MIT-GATEWAYS - Massachusetts Institute of Technology"
	expected[397142] = "FORT-COLLINS-CONNEXION - City of Fort Collins"
	expected[397143] = "NEPTUNE-NETWORKS - Neptune Networks"
	expected[397144] = "VOLTERRA-02 - Volterra Inc"

	body, _ := ioutil.ReadFile("testdata/asn.txt")
	actual, _ := parse(body)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("\nExpected: %v\nReceived: %v", expected, actual)
	}
}
