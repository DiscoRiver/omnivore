package group

import (
	"math/rand"
	"testing"
)

var (
	byt = makeTestByte()
)

func makeTestByte() []byte {
	b := make([]byte, 1000000)
	rand.Read(b)
	return b
}

func BenchmarkEncodeByteSliceToUint32(b *testing.B) {
	for n := 0; n < b.N; n++ {
		EncodeByteSliceToUint32(byt)
	}
}

func BenchmarkEncodeByteSliceToSha1(b *testing.B) {
	for n := 0; n < b.N; n++ {
		EncodeByteSliceToSha1(byt)
	}
}

func BenchmarkEncodeByteSliceToMD5(b *testing.B) {
	for n := 0; n < b.N; n++ {
		EncodeByteSliceToMD5(byt)
	}
}

func BenchmarkEncodeByteSliceToMD4(b *testing.B) {
	for n := 0; n < b.N; n++ {
		EncodeByteSliceToMD4(byt)
	}
}

func BenchmarkEncodeByteSliceToCRC32(b *testing.B) {
	for n := 0; n < b.N; n++ {
		EncodeByteSliceToCRC32(byt)
	}
}
