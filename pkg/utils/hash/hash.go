package hash

import (
	"crypto/md5"
	"encoding/binary"
)

func Hash16(s string) uint16 {
	hashb := md5.Sum([]byte(s))
	return binary.LittleEndian.Uint16(hashb[:2])
}

func Hash32(s string) uint32 {
	hashb := md5.Sum([]byte(s))
	return binary.LittleEndian.Uint32(hashb[:4])
}

func Hash64(s string) uint64 {
	hashb := md5.Sum([]byte(s))
	return binary.LittleEndian.Uint64(hashb[:8])
}
