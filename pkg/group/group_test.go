package group

import (
	"sync"
	"testing"
)

func TestValueGrouping_AddNewGroup(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	err := vg.AddNewGroup(i)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if _, ok := vg.Map[i.EncodedValue]; !ok {
		t.Logf("Map key not present for expect value: %d", i.EncodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddMemberCreate_Present(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))
	i2 := NewIdentifyingPair("host2", []byte("Hello, World"))

	vg := NewValueGrouping()

	err := vg.AddNewGroup(i)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	vg.AddMemberCreate(i2)

	if members, _ := vg.Map[i.EncodedValue]; members[1] != "host2" {
		t.Logf("Expected %s to be present in map slice.", i2.Key)
		t.Fail()
	}
}

func TestValueGrouping_AddMemberCreate_NotPresent(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	vg.AddMemberCreate(i)

	if members, ok := vg.Map[i.EncodedValue]; !ok && len(members) != 1 {
		t.Logf("Map key not present for expect value: %d", i.EncodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddMemberCreate_Concurrent(t *testing.T) {
	var iden []*IdentifyingPair
	iden = append(iden, NewIdentifyingPair("host", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host2", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host3", []byte("Hello, Worlds")))
	iden = append(iden, NewIdentifyingPair("host4", []byte("Hello, Worlds")))

	vg := NewValueGrouping()

	AddMemberCreateFunc := func(i *IdentifyingPair, wg *sync.WaitGroup){
		vg.AddMemberCreate(i)
		wg.Done()
	}

	var wg sync.WaitGroup
	for i := range iden {
		wg.Add(1)
		go AddMemberCreateFunc(iden[i], &wg)
	}
	wg.Wait()


	if members, ok := vg.Map[iden[0].EncodedValue]; !ok && len(members) != 2 {
		t.Logf("Map key not present for expect value: %d", iden[0].EncodedValue)
		t.Fail()
	}

	if members, ok := vg.Map[iden[2].EncodedValue]; !ok && len(members) != 2 {
		t.Logf("Map key not present for expect value: %d", iden[0].EncodedValue)
		t.Fail()
	}

}
