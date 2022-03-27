package ossh

import (
	"testing"

	"github.com/discoriver/massh"
)

var (
	mockResult []massh.Result

	sampleResult1 = massh.Result{
		Host: "host1",
	}

	sampleResult2 = massh.Result{
		Host: "host2",
	}
)

func TestPopulateResultsMap_IntegrationWorkflow(t *testing.T) {
	mockResult = append(mockResult, sampleResult1, sampleResult2)
	s := StreamCycle{}
	s.Initialise()

	ch := make(chan massh.Result)

	for i := range mockResult {
		i := i
		go func() { ch <- mockResult[i] }()
	}

	go func() {
		for {
			select {
			case <-s.HostsResultChan:
			}
		}
	}()

	s.populateResultsMap(ch, len(mockResult))

	if s.NumHostsInit != len(mockResult) {
		t.Errorf("number of hosts expected %v, got %v", len(mockResult), s.NumHostsInit)
	}
}

func TestGetSortedHostMapKeys(t *testing.T) {
	m := map[string]struct{}{}

	m["host1"] = struct{}{}
	m["host2"] = struct{}{}

	expected := 2
	if len(GetSortedHostMapKeys(m)) != expected {
		t.Errorf("expected %d keys from map, got %d", expected, len(GetSortedHostMapKeys(m)))
	}
}
