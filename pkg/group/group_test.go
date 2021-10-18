package group

import "testing"

func TestValueGrouping_AddNewGroup(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	err := vg.AddNewGroup(i)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if _, ok := vg.Map[i.hash]; !ok {
		t.Logf("Map key not present for expect value: %d", i.hash)
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

	if members, _ := vg.Map[i.hash]; members[1] != "host2" {
		t.Logf("Expected %s to be present in map slice.", i2.Key)
		t.Fail()
	}
}

func TestValueGrouping_AddMemberCreate_NotPresent(t *testing.T) {
	i := NewIdentifyingPair("host", []byte("Hello, World"))

	vg := NewValueGrouping()

	vg.AddMemberCreate(i)

	if members, ok := vg.Map[i.hash]; !ok && len(members) != 1 {
		t.Logf("Map key not present for expect value: %d", i.hash)
		t.Fail()
	}
}
