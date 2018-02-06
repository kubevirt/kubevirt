// Package xxHash64 implements the very fast xxHash hashing algorithm (64 bits version).
// (https://github.com/Cyan4973/xxHash/)
package xxHash64

import "hash"

const (
	prime64_1 = 11400714785074694791
	prime64_2 = 14029467366897019727
	prime64_3 = 1609587929392839161
	prime64_4 = 9650029242287828579
	prime64_5 = 2870177450012600261
)

type xxHash struct {
	seed     uint64
	v1       uint64
	v2       uint64
	v3       uint64
	v4       uint64
	totalLen uint64
	buf      [32]byte
	bufused  int
}

// New returns a new Hash64 instance.
func New(seed uint64) hash.Hash64 {
	xxh := &xxHash{seed: seed}
	xxh.Reset()
	return xxh
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (xxh xxHash) Sum(b []byte) []byte {
	h64 := xxh.Sum64()
	return append(b, byte(h64), byte(h64>>8), byte(h64>>16), byte(h64>>24), byte(h64>>32), byte(h64>>40), byte(h64>>48), byte(h64>>56))
}

// Reset resets the Hash to its initial state.
func (xxh *xxHash) Reset() {
	xxh.v1 = xxh.seed + prime64_1 + prime64_2
	xxh.v2 = xxh.seed + prime64_2
	xxh.v3 = xxh.seed
	xxh.v4 = xxh.seed - prime64_1
	xxh.totalLen = 0
	xxh.bufused = 0
}

// Size returns the number of bytes returned by Sum().
func (xxh *xxHash) Size() int {
	return 8
}

// BlockSize gives the minimum number of bytes accepted by Write().
func (xxh *xxHash) BlockSize() int {
	return 1
}

// Write adds input bytes to the Hash.
// It never returns an error.
func (xxh *xxHash) Write(input []byte) (int, error) {
	n := len(input)
	m := xxh.bufused

	xxh.totalLen += uint64(n)

	r := len(xxh.buf) - m
	if n < r {
		copy(xxh.buf[m:], input)
		xxh.bufused += len(input)
		return n, nil
	}

	p := 0
	if m > 0 {
		// some data left from previous update
		copy(xxh.buf[xxh.bufused:], input[:r])
		xxh.bufused += len(input) - r

		// fast rotl(31)
		xxh.v1 = rol31(xxh.v1+u64(xxh.buf[:])*prime64_2) * prime64_1
		xxh.v2 = rol31(xxh.v2+u64(xxh.buf[8:])*prime64_2) * prime64_1
		xxh.v3 = rol31(xxh.v3+u64(xxh.buf[16:])*prime64_2) * prime64_1
		xxh.v4 = rol31(xxh.v4+u64(xxh.buf[24:])*prime64_2) * prime64_1
		p = r
		xxh.bufused = 0
	}

	// Causes compiler to work directly from registers instead of stack:
	v1, v2, v3, v4 := xxh.v1, xxh.v2, xxh.v3, xxh.v4
	for n := n - 32; p <= n; p += 32 {
		sub := input[p:][:32] //BCE hint for compiler
		v1 = rol31(v1+u64(sub[:])*prime64_2) * prime64_1
		v2 = rol31(v2+u64(sub[8:])*prime64_2) * prime64_1
		v3 = rol31(v3+u64(sub[16:])*prime64_2) * prime64_1
		v4 = rol31(v4+u64(sub[24:])*prime64_2) * prime64_1
	}
	xxh.v1, xxh.v2, xxh.v3, xxh.v4 = v1, v2, v3, v4

	copy(xxh.buf[xxh.bufused:], input[p:])
	xxh.bufused += len(input) - p

	return n, nil
}

// Sum64 returns the 64bits Hash value.
func (xxh *xxHash) Sum64() uint64 {
	var h64 uint64
	if xxh.totalLen >= 32 {
		h64 = rol1(xxh.v1) + rol7(xxh.v2) + rol12(xxh.v3) + rol18(xxh.v4)

		xxh.v1 *= prime64_2
		xxh.v2 *= prime64_2
		xxh.v3 *= prime64_2
		xxh.v4 *= prime64_2

		h64 = (h64^(rol31(xxh.v1)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(xxh.v2)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(xxh.v3)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(xxh.v4)*prime64_1))*prime64_1 + prime64_4

		h64 += xxh.totalLen
	} else {
		h64 = xxh.seed + prime64_5 + xxh.totalLen
	}

	p := 0
	n := xxh.bufused
	for n := n - 8; p <= n; p += 8 {
		h64 ^= rol31(u64(xxh.buf[p:p+8])*prime64_2) * prime64_1
		h64 = rol27(h64)*prime64_1 + prime64_4
	}
	if p+4 <= n {
		sub := xxh.buf[p : p+4]
		h64 ^= uint64(u32(sub)) * prime64_1
		h64 = rol23(h64)*prime64_2 + prime64_3
		p += 4
	}
	for ; p < n; p++ {
		h64 ^= uint64(xxh.buf[p]) * prime64_5
		h64 = rol11(h64) * prime64_1
	}

	h64 ^= h64 >> 33
	h64 *= prime64_2
	h64 ^= h64 >> 29
	h64 *= prime64_3
	h64 ^= h64 >> 32

	return h64
}

// Checksum returns the 64bits Hash value.
func Checksum(input []byte, seed uint64) uint64 {
	n := len(input)
	var h64 uint64

	if n >= 32 {
		v1 := seed + prime64_1 + prime64_2
		v2 := seed + prime64_2
		v3 := seed
		v4 := seed - prime64_1
		p := 0
		for n := n - 32; p <= n; p += 32 {
			sub := input[p:][:32] //BCE hint for compiler
			v1 = rol31(v1+u64(sub[:])*prime64_2) * prime64_1
			v2 = rol31(v2+u64(sub[8:])*prime64_2) * prime64_1
			v3 = rol31(v3+u64(sub[16:])*prime64_2) * prime64_1
			v4 = rol31(v4+u64(sub[24:])*prime64_2) * prime64_1
		}

		h64 = rol1(v1) + rol7(v2) + rol12(v3) + rol18(v4)

		v1 *= prime64_2
		v2 *= prime64_2
		v3 *= prime64_2
		v4 *= prime64_2

		h64 = (h64^(rol31(v1)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(v2)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(v3)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(v4)*prime64_1))*prime64_1 + prime64_4

		h64 += uint64(n)

		input = input[p:]
		n -= p
	} else {
		h64 = seed + prime64_5 + uint64(n)
	}

	p := 0
	for n := n - 8; p <= n; p += 8 {
		sub := input[p : p+8]
		h64 ^= rol31(u64(sub)*prime64_2) * prime64_1
		h64 = rol27(h64)*prime64_1 + prime64_4
	}
	if p+4 <= n {
		sub := input[p : p+4]
		h64 ^= uint64(u32(sub)) * prime64_1
		h64 = rol23(h64)*prime64_2 + prime64_3
		p += 4
	}
	for ; p < n; p++ {
		h64 ^= uint64(input[p]) * prime64_5
		h64 = rol11(h64) * prime64_1
	}

	h64 ^= h64 >> 33
	h64 *= prime64_2
	h64 ^= h64 >> 29
	h64 *= prime64_3
	h64 ^= h64 >> 32

	return h64
}

func u64(buf []byte) uint64 {
	// go compiler recognizes this pattern and optimizes it on little endian platforms
	return uint64(buf[0]) | uint64(buf[1])<<8 | uint64(buf[2])<<16 | uint64(buf[3])<<24 | uint64(buf[4])<<32 | uint64(buf[5])<<40 | uint64(buf[6])<<48 | uint64(buf[7])<<56
}

func u32(buf []byte) uint32 {
	return uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
}

func rol1(u uint64) uint64 {
	return u<<1 | u>>63
}

func rol7(u uint64) uint64 {
	return u<<7 | u>>57
}

func rol11(u uint64) uint64 {
	return u<<11 | u>>53
}

func rol12(u uint64) uint64 {
	return u<<12 | u>>52
}

func rol18(u uint64) uint64 {
	return u<<18 | u>>46
}

func rol23(u uint64) uint64 {
	return u<<23 | u>>41
}

func rol27(u uint64) uint64 {
	return u<<27 | u>>37
}
func rol31(u uint64) uint64 {
	return u<<31 | u>>33
}
