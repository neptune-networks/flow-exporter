package asndb

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchWithValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testDataFile := "testdata/asn.txt"
		testData, err := ioutil.ReadFile(testDataFile)
		if err != nil {
			t.Errorf("Error reading test data: %s", testDataFile)
		}

		w.Write(testData)
	}))

	defer server.Close()

	db := ASNDB{
		url:       server.URL,
		asnFormat: `([\d]+)\s+(.*),\s(\w{2})`,
		http:      &http.Client{},
	}

	actual, err := db.Fetch()
	if err != nil {
		t.Errorf("Expected to receive response from server, instead received: %s", err.Error())
	}

	expected := make(map[int]string)
	expected[0] = "-Reserved AS-"
	expected[1] = "LVLT-1 - Level 3 Parent, LLC"
	expected[2] = "UDEL-DCN - University of Delaware"
	expected[3] = "MIT-GATEWAYS - Massachusetts Institute of Technology"
	expected[397142] = "FORT-COLLINS-CONNEXION - City of Fort Collins"
	expected[397143] = "NEPTUNE-NETWORKS - Neptune Networks"
	expected[397144] = "VOLTERRA-02 - Volterra Inc"

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("\nExpected: %v\nReceived: %v", expected, actual)
	}
}

func TestFetchWithUnavailableServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	db := ASNDB{
		url:       server.URL,
		asnFormat: `([\d]+)\s+(.*),\s(\w{2})`,
		http:      &http.Client{},
	}

	actual, _ := db.Fetch()
	if actual != nil {
		t.Errorf("Did not expect to receive valid response from server, received: %v", actual)
	}
}

func TestFetchWithInvalidData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testDataFile := "testdata/invalid_asn.txt"
		testData, err := ioutil.ReadFile(testDataFile)
		if err != nil {
			t.Errorf("Error reading test data: %s", testDataFile)
		}

		w.Write(testData)
	}))

	defer server.Close()

	db := ASNDB{
		url:       server.URL,
		asnFormat: `([\d]+)`,
		http:      &http.Client{},
	}

	actual, err := db.Fetch()
	if err != nil {
		t.Errorf("Expected to receive response from server, instead received: %s", err.Error())
	}

	expected := make(map[int]string)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected empty ASN database, instead received: %v", actual)
	}
}
