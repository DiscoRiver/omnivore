// Package group provides a method for taking an identifier and value pair, comparing them with other pairs, and then grouping identical values.
package group

import (
	"fmt"
	"sort"
	"sync"
)

// ValueGrouping contains value/member groupings, and should only be written to by using the AddToGroup method.
type ValueGrouping struct {
	// Representing a byte slice as an encoded value for easy equality check. String slice of members.
	EncodedValueGroup map[string][]string
	// Preserve original byte slice
	EncodedValueToOriginal map[string][]byte

	mu     sync.Mutex
	Update chan struct{}
}

type IdentifyingPair struct {
	Key string
	// WARN: Potentially memory intensive if returning lots of data from ssh command. Might want to consider temp files
	// if number of bytes exceeds a limit.
	Value        []byte
	encodedValue string

	mu sync.Mutex
}

// Creates a new IdentifyingPair, using the provide Key and Value pair. This can then be added to a new ValueGrouping, or otherwise processed.
func NewIdentifyingPair(Key string, Value []byte) *IdentifyingPair {
	return &IdentifyingPair{
		Key:          Key,
		Value:        Value,
		encodedValue: EncodeByteSliceToMD5(Value),
	}
}

// Initialised a new ValueGrouping used to populate
func NewValueGrouping() *ValueGrouping {
	vg := &ValueGrouping{
		EncodedValueGroup:      map[string][]string{},
		EncodedValueToOriginal: map[string][]byte{},
		Update:                 make(chan struct{}),
	}
	return vg
}

func GetSortedGroupMapKeys(m map[string][]string) []string {
	var keys []string
	for k, _ := range m {
		if k != "" {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)
	return keys
}

// AddToGroup creates or adds to an EncodedValueGroup. If an entry already exists for the encoded value provided within
// IdentifyingPair, the additional members will be added. This should be considered the only function for adding groups
// and members to an EncodedValueGroup.
func (v *ValueGrouping) AddToGroup(i *IdentifyingPair) {
	v.mu.Lock()
	i.mu.Lock()
	defer func() {
		v.mu.Unlock()
		i.mu.Unlock()
	}()

	if i.encodedValue == "" {
		i.encodedValue = EncodeByteSliceToMD5(i.Value)
	}

	if _, ok := v.EncodedValueGroup[i.encodedValue]; ok {
		v.addMembersToExistingGroup(i)
		return
	}

	v.EncodedValueGroup[i.encodedValue] = []string{i.Key}
	v.EncodedValueToOriginal[i.encodedValue] = i.Value

	v.Update <- struct{}{}
	return
}

func (v *ValueGrouping) addMembersToExistingGroup(i *IdentifyingPair) {
	v.EncodedValueGroup[i.encodedValue] = append(v.EncodedValueGroup[i.encodedValue], i.Key)
	v.Update <- struct{}{}
}

func (v *ValueGrouping) GetMembers(hash string) ([]string, error) {
	v.mu.Lock()
	defer func() { v.mu.Unlock() }()

	if members, ok := v.EncodedValueGroup[hash]; !ok {
		return nil, fmt.Errorf("value entry does not exist, no members to return")
	} else {
		return members, nil
	}
}

func (v *ValueGrouping) GetValue(hash string) ([]byte, error) {
	v.mu.Lock()
	defer func() { v.mu.Unlock() }()

	if value, ok := v.EncodedValueToOriginal[hash]; !ok {
		return nil, fmt.Errorf("encodedValue entry does not exist, no value to return")
	} else {
		return value, nil
	}
}
