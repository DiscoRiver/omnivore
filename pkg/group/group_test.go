package group

import (
	"sync"
	"testing"
)

func TestValueGrouping_AddToGroup_UnitWorkflow(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	// Make sure we're always reading for updates or it'll block.
	go checkForUpdates(vg)

	vg.AddToGroup(i)

	if _, ok := vg.EncodedValueGroup[i.encodedValue]; !ok {
		t.Logf("EncodedValueGroup key not present for expect value: %s", i.encodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_Present_UnitWorkflow(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))
	i2 := NewIdentifyingPair("host2", []byte("Hello, World"))

	vg := NewValueGrouping()

	// Make sure we're always reading for updates or it'll block.
	go checkForUpdates(vg)

	vg.AddToGroup(i)
	vg.AddToGroup(i2)

	if members, _ := vg.EncodedValueGroup[i.encodedValue]; members[1] != "host2" {
		t.Logf("Expected %s to be present in map slice.", i2.Key)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_NotPresent_UnitWorkflow(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	// Make sure we're always reading for updates or it'll block.
	go checkForUpdates(vg)

	vg.AddToGroup(i)

	if members, ok := vg.EncodedValueGroup[i.encodedValue]; !ok && len(members) != 1 {
		t.Logf("EncodedValueGroup key not present for expect value: %s", i.encodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_Concurrent_UnitWorkflow(t *testing.T) {
	var iden []*IdentifyingPair
	iden = append(iden, NewIdentifyingPair("host", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host2", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host3", []byte("Hello, Worlds")))
	iden = append(iden, NewIdentifyingPair("host4", []byte("Hello, Worlds")))

	vg := NewValueGrouping()

	// Make sure we're always reading for updates or it'll block.
	go checkForUpdates(vg)

	AddMemberCreateFunc := func(i *IdentifyingPair, wg *sync.WaitGroup) {
		vg.AddToGroup(i)
		wg.Done()
	}

	var wg sync.WaitGroup
	for i := range iden {
		wg.Add(1)
		go AddMemberCreateFunc(iden[i], &wg)
	}
	wg.Wait()

	if members, ok := vg.EncodedValueGroup[iden[0].encodedValue]; !ok && len(members) != 2 {
		t.Logf("EncodedValueGroup key not present for expect value: %s", iden[0].encodedValue)
		t.Fail()
	}

	if members, ok := vg.EncodedValueGroup[iden[2].encodedValue]; !ok && len(members) != 2 {
		t.Logf("EncodedValueGroup key not present for expect value: %s", iden[0].encodedValue)
		t.Fail()
	}
}

// Test when IdentifyingPair has a zero encodedValue due to bad initialisation.
func TestValueGrouping_AddToGroup_Uninitialised_UnitWorkflow(t *testing.T) {
	i := &IdentifyingPair{
		Key:          "host",
		Value:        []byte("Hello, World"),
		encodedValue: "",
		mu:           sync.Mutex{},
	}

	vg := NewValueGrouping()

	// Make sure we're always reading for updates or it'll block.
	go checkForUpdates(vg)

	vg.AddToGroup(i)

	if members, ok := vg.EncodedValueGroup[i.encodedValue]; !ok && members[0] != i.Key {
		t.Logf("EncodedValueGroup key not present for expect value: %s", i.encodedValue)
		t.Fail()
	}
}

func checkForUpdates(vg *ValueGrouping) {
	for {
		select {
		case <-vg.Update:
		}
	}
}
