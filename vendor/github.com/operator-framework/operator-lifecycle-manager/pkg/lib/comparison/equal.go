package comparison

import (
	// "fmt"

	"github.com/mitchellh/hashstructure"
)

// Equalitor describes an algorithm for equivalence between two data structures.
type Equalitor interface {
	Equal(a, b interface{}) bool
}

// EqualFunc is a function that implements Equalitor.
type EqualFunc func(a, b interface{}) bool

// Equal allows an EqualFunc to implement Equalitor.
func (e EqualFunc) Equal(a, b interface{}) bool {
	return e(a, b)
}

// NewHashEqualitor returns an EqualFunc that returns true if the hashes of the given
// arguments are equal, false otherwise.
//
// This function panics if an error is encountered generating a hash for either argument.
func NewHashEqualitor() EqualFunc {
	return func(a, b interface{}) bool {
		hashA, err := hashstructure.Hash(a, nil)
		if err != nil {
			panic(err.Error())
		}

		hashB, err := hashstructure.Hash(b, nil)
		if err != nil {
			panic(err.Error())
		}

		// fmt.Printf("hashA: %d, hashB: %d\n", hashA, hashB)

		return hashA == hashB
	}
}
