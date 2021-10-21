// Package group provides a method for taking an identifier and value pair, comparing them with other pairs, and then grouping identical values.
package group

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/md4"
	"sync"
)

// ValueGrouping contains value/member groupings, and should only be written to by using the AddToGroup method.
type ValueGrouping struct {
	// Representing a byte slice as an encoded value for easy equality check. String slice of members.
	EncodedValueGroup map[uint32][]string
	// Preserve original byte slice
	EncodedValueToOriginal map[uint32][]byte

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
		EncodedValueGroup:      map[uint32][]string{},
		EncodedValueToOriginal: map[uint32][]byte{},
	}
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

	if i.EncodedValue == 0 {
		i.EncodedValue = EncodeByteSliceToUint32(i.Value)
	}

	if _, ok := v.EncodedValueGroup[i.EncodedValue]; ok {
		v.addMembersToExistingGroup(i)
		return
	}

	v.EncodedValueGroup[i.EncodedValue] = []string{i.Key}
	v.EncodedValueToOriginal[i.EncodedValue] = i.Value

	return
}

func (v *ValueGrouping) addMembersToExistingGroup(i *IdentifyingPair) {
	v.EncodedValueGroup[i.EncodedValue] = append(v.EncodedValueGroup[i.EncodedValue], i.Key)
}

func (v *ValueGrouping) GetMembers(hash uint32) ([]string, error) {
	v.mu.Lock()
	defer func(){ v.mu.Unlock() }()

	if members, ok := v.EncodedValueGroup[hash]; !ok {
		return nil, fmt.Errorf("value entry does not exist, no members to return")
	} else {
		return members, nil
	}
}

func (v *ValueGrouping) GetValue(hash uint32) ([]byte, error) {
	v.mu.Lock()
	defer func(){ v.mu.Unlock() }()

	if value, ok := v.EncodedValueToOriginal[hash]; !ok {
		return nil, fmt.Errorf("EncodedValue entry does not exist, no value to return")
	} else {
		return value, nil
	}
}

func EncodeByteSliceToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func EncodeByteSliceToSha1(byt []byte) string {
	h := sha1.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}

func EncodeByteSliceToMD5(byt []byte) string {
	h := md5.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}

func EncodeByteSliceToMD4(byt []byte) string {
	h := md4.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}