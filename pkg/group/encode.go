package group

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"golang.org/x/crypto/md4"
	"hash/crc32"
)

func EncodeByteSliceToMD5(byt []byte) string {
	h := md5.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}

/*
The following encoding funcs are here for experimentation and for the benchmarks in group_test.go - Since MD5 is the
quickest algorithm right now, we're using that, but we may want to explore other options in the future.
*/

// Only hashes first 4 bytes. Probably not useful.
func EncodeByteSliceToUint32(b []byte) uint32 {
	if len(b) <= 4 && len(b) != 0 {
		b = append(b, []byte(encodePadding)...)
	}
	return binary.BigEndian.Uint32(b)
}

func EncodeByteSliceToSha1(byt []byte) string {
	h := sha1.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}

func EncodeByteSliceToMD4(byt []byte) string {
	h := md4.New()
	h.Write(byt)
	return hex.EncodeToString(h.Sum(nil))
}

func EncodeByteSliceToCRC32(byt []byte) uint32 {
	tab := crc32.MakeTable(0xD5828281)
	return crc32.Checksum(byt, tab)
}
