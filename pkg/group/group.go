// Package group provides a method for taking an identifier and value pair, comparing them with other pairs, and then grouping identical values.
package group

import (
	"encoding/binary"
	"fmt"
)

type ValueGrouping struct {
	// Representing a byte slice as hash value for equality check. String slice of members
	Map map[uint32][]string
	// Preserve original byte slice
	Value map[uint32][]byte
}


type IdentifyingPair struct {
	Key string
	// WARN: Potentially memory intensive if returning lots of data from ssh command.
	Value []byte

	// Unexported
	hash uint32
}

func NewIdentifyingPair(Key string, Value []byte) *IdentifyingPair {
	return &IdentifyingPair{
		Key:   Key,
		Value: Value,
		hash:  ComputeUint32Hash(Value),
	}
}

func NewValueGrouping() *ValueGrouping {
	return &ValueGrouping{
		Map:   map[uint32][]string{},
		Value: map[uint32][]byte{},
	}
}

func (v *ValueGrouping) AddNewGroup(i *IdentifyingPair) error {
	if i.hash == 0 {
		i.hash = ComputeUint32Hash(i.Value)
	}

	if _, ok := v.Map[i.hash]; ok {
		return fmt.Errorf("value group already exists")
	}

	v.Map[i.hash] = []string{i.Key}
	v.Value[i.hash] = i.Value
	return nil
}

// AddMember adds members to the value group, if hash doesn't currently exist, it will be created.
func (v *ValueGrouping) AddMemberCreate(i *IdentifyingPair) {
	if i.hash == 0 {
		i.hash = ComputeUint32Hash(i.Value)
	}

	if _, ok := v.Map[i.hash]; !ok {
		v.AddNewGroup(i)
	} else {
		v.Map[i.hash] = append(v.Map[i.hash], i.Key)
	}
}

func (v *ValueGrouping) GetMembers(hash uint32) ([]string, error) {
	if members, ok := v.Map[hash]; !ok {
		return nil, fmt.Errorf("value entry does not exist, no members to return")
	} else {
		return members, nil
	}
}

func (v *ValueGrouping) GetValue(hash uint32) ([]byte, error) {
	if value, ok := v.Value[hash]; !ok {
		return nil, fmt.Errorf("hash entry does not exist, no value to return")
	} else {
		return value, nil
	}
}

func ComputeUint32Hash(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

// Grouping for short output groups. We can perform grouping here based on an exact match, or by a Levenshtein
// distance value. We're typically expecting single-line output from these commands. Some examples could for commands
// that would fall into this category could be "date", "lsb_release -v", or "uname".
type ShortOutputGroup struct {
	Hosts map[string]int

	Output []byte
	// Output length
	len int

	MaxDeviation int
}