package ssh

import (
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
	"testing"
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

func TestPopulateResultsMap(t *testing.T) {
	log.InitTestLogger()

	mockResult = append(mockResult, sampleResult1, sampleResult2)
	s := StreamCycle{}
	s.Initialise()
	ch := make(chan massh.Result)

	for i := range mockResult {
		i := i
		go func() { ch <- mockResult[i] }()
	}

	err := s.populateResultsMap(ch, len(mockResult))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if len(s.HostsResultMap) == len(mockResult) {
		if _, ok := s.HostsResultMap["host1"]; !ok {
			t.FailNow()
		}
	} else {
		t.Errorf("number of hosts expected %v, got %v", len(mockResult), len(s.HostsResultMap))
	}
}
