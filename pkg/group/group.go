// Package group provides a method for taking an identifier and value pair, comparing them with other pairs, and then grouping identical values.
package group

import (
	"encoding/binary"
	"fmt"
	"sync"
)

type ValueGrouping struct {
	// Representing a byte slice as EncodedValue value for equality check. String slice of members
	Map map[uint32][]string
	// Preserve original byte slice
	Value map[uint32][]byte

	mu sync.Mutex
}

type IdentifyingPair struct {
	Key string
	// WARN: Potentially memory intensive if returning lots of data from ssh command.
	Value []byte
	EncodedValue uint32

	mu sync.Mutex
}

func NewIdentifyingPair(Key string, Value []byte) *IdentifyingPair {
	return &IdentifyingPair{
		Key:          Key,
		Value:        Value,
		EncodedValue: EncodeByteSliceToUint32(Value),
	}
}

func NewValueGrouping() *ValueGrouping {
	return &ValueGrouping{
		Map:   map[uint32][]string{},
		Value: map[uint32][]byte{},
	}
}

func (v *ValueGrouping) AddNewGroup(i *IdentifyingPair) error {
	if i.EncodedValue == 0 {
		i.EncodedValue = EncodeByteSliceToUint32(i.Value)
	}

	if _, ok := v.Map[i.EncodedValue]; ok {
		return fmt.Errorf("value group already exists")
	}

	v.Map[i.EncodedValue] = []string{i.Key}
	v.Value[i.EncodedValue] = i.Value
	return nil
}

// AddMember adds members to the value group, if EncodedValue doesn't currently exist, it will be created.
func (v *ValueGrouping) AddMemberCreate(i *IdentifyingPair) {
	v.mu.Lock()
	i.mu.Lock()
	defer func() {
		v.mu.Unlock()
		i.mu.Unlock()
	}()

	if i.EncodedValue == 0 {
		i.EncodedValue = EncodeByteSliceToUint32(i.Value)
	}

	if _, ok := v.Map[i.EncodedValue]; !ok {
		v.AddNewGroup(i)
	} else {
		v.Map[i.EncodedValue] = append(v.Map[i.EncodedValue], i.Key)
	}
}

func (v *ValueGrouping) GetMembers(hash uint32) ([]string, error) {
	v.mu.Lock()
	defer func(){ v.mu.Unlock() }()

	if members, ok := v.Map[hash]; !ok {
		return nil, fmt.Errorf("value entry does not exist, no members to return")
	} else {
		return members, nil
	}
}

func (v *ValueGrouping) GetValue(hash uint32) ([]byte, error) {
	v.mu.Lock()
	defer func(){ v.mu.Unlock() }()

	if value, ok := v.Value[hash]; !ok {
		return nil, fmt.Errorf("EncodedValue entry does not exist, no value to return")
	} else {
		return value, nil
	}
}

func EncodeByteSliceToUint32(b []byte) uint32 {
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