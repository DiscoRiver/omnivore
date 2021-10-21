package group

import (
	"sync"
	"testing"
)

func TestValueGrouping_AddToGroup(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	vg.AddToGroup(i)

	if _, ok := vg.EncodedValueGroup[i.EncodedValue]; !ok {
		t.Logf("EncodedValueGroup key not present for expect value: %d", i.EncodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_Present(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))
	i2 := NewIdentifyingPair("host2", []byte("Hello, World"))

	vg := NewValueGrouping()

	vg.AddToGroup(i)
	vg.AddToGroup(i2)

	if members, _ := vg.EncodedValueGroup[i.EncodedValue]; members[1] != "host2" {
		t.Logf("Expected %s to be present in map slice.", i2.Key)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_NotPresent(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	vg.AddToGroup(i)

	if members, ok := vg.EncodedValueGroup[i.EncodedValue]; !ok && len(members) != 1 {
		t.Logf("EncodedValueGroup key not present for expect value: %d", i.EncodedValue)
		t.Fail()
	}
}

func TestValueGrouping_AddToGroup_Concurrent(t *testing.T) {
	var iden []*IdentifyingPair
	iden = append(iden, NewIdentifyingPair("host", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host2", []byte("Hello, World")))
	iden = append(iden, NewIdentifyingPair("host3", []byte("Hello, Worlds")))
	iden = append(iden, NewIdentifyingPair("host4", []byte("Hello, Worlds")))

	vg := NewValueGrouping()

	AddMemberCreateFunc := func(i *IdentifyingPair, wg *sync.WaitGroup){
		vg.AddToGroup(i)
		wg.Done()
	}

	var wg sync.WaitGroup
	for i := range iden {
		wg.Add(1)
		go AddMemberCreateFunc(iden[i], &wg)
	}
	wg.Wait()


	if members, ok := vg.EncodedValueGroup[iden[0].EncodedValue]; !ok && len(members) != 2 {
		t.Logf("EncodedValueGroup key not present for expect value: %d", iden[0].EncodedValue)
		t.Fail()
	}

	if members, ok := vg.EncodedValueGroup[iden[2].EncodedValue]; !ok && len(members) != 2 {
		t.Logf("EncodedValueGroup key not present for expect value: %d", iden[0].EncodedValue)
		t.Fail()
	}
}

// Test when IdentifyingPair has a zero EncodedValue due to bad initialisation.
func TestValueGrouping_AddToGroup_Uninitialised(t *testing.T) {
	i := &IdentifyingPair{
		Key:          "host",
		Value:        []byte("Hello, World"),
		EncodedValue: 0,
		mu:           sync.Mutex{},
	}

	vg := NewValueGrouping()

	vg.AddToGroup(i)

	if _, ok := vg.EncodedValueGroup[i.EncodedValue]; !ok {
		t.Logf("EncodedValueGroup key not present for expect value: %d", i.EncodedValue)
		t.Fail()
	}
}

func BenchmarkEncodeByteSliceToUint32(b *testing.B) {
	byt := []byte("Hello, World")

	for n := 0; n < b.N; n++ {
		EncodeByteSliceToUint32(byt)
	}
}

func BenchmarkEncodeByteSliceToSha1(b *testing.B) {
	byt := []byte("Hello, World")

	for n := 0; n < b.N; n++ {
		EncodeByteSliceToSha1(byt)
	}
}

func BenchmarkEncodeByteSliceToMD5(b *testing.B) {
	byt := []byte("Hello, World")

	for n := 0; n < b.N; n++ {
		EncodeByteSliceToMD5(byt)
	}
}

func BenchmarkEncodeByteSliceToMD4(b *testing.B) {
	byt := []byte("Hello, World")

	for n := 0; n < b.N; n++ {
		EncodeByteSliceToMD4(byt)
	}
}
