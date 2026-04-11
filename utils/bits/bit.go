package bits

import (
	"bytes"
	"fmt"
)

// 根据平台自动确定uint的位数：32位平台为32，64位平台为64
const (
	bitsPerWord = 32 << (^uint(0) >> 63) // 32 or 64
	bitShift    = 5 + (^uint(0) >> 63)   // 5 for 32-bit, 6 for 64-bit
	bitMask     = bitsPerWord - 1        // 31 or 63
)

type IntSet struct {
	words []uint
}

func (s *IntSet) Add(x int) {
	word := x >> bitShift
	bit := uint(x) & bitMask
	for word >= len(s.words) {
		s.words = append(s.words, 0)
	}
	s.words[word] |= 1 << bit
}

func (s *IntSet) UnionWith(o *IntSet) {
	for i, word := range o.words {
		if i < len(s.words) {
			s.words[i] |= word
			continue
		}
		s.words = append(s.words, word)
	}
}

func (s *IntSet) Has(x int) bool {
	word := x >> bitShift
	bit := uint(x) & bitMask
	return word < len(s.words) && s.words[word]&(1<<bit) != 0
}

// String returns a string representation of the set as a comma-separated list of set bit positions,
// enclosed in curly braces. For example, a set containing bits 1, 3, and 5 would return "{1 3 5}".
// An empty set returns "{}".
func (s *IntSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < bitsPerWord; j++ {
			if word&(1<<uint(j)) != 0 {
				if buf.Len() > len("{") {
					buf.WriteByte(' ')
				}
				buf.WriteString(fmt.Sprintf("%d", bitsPerWord*i+j))
			}
		}
	}
	buf.WriteByte('}')
	return buf.String()
}

func (s *IntSet) Len() int {
	count := 0
	for _, word := range s.words {
		for word != 0 {
			count++
			word &= word - 1 // clear the least significant bit set
		}
	}
	return count
}

func (s *IntSet) Remove(x int) {
	word := x >> bitShift
	bit := uint(x) & bitMask
	if word < len(s.words) {
		s.words[word] &^= 1 << bit
	}
}

func (s *IntSet) clear() {
	s.words = nil
}

func (s *IntSet) Copy() *IntSet {
	t := &IntSet{}
	t.words = make([]uint, len(s.words))
	copy(t.words, s.words)
	return t
}

func (s *IntSet) AddAll(xs ...int) {
	for _, x := range xs {
		s.Add(x)
	}
}

func (s *IntSet) IntersectWith(t *IntSet) {
	minLen := min(len(t.words), len(s.words))
	for i := range minLen {
		s.words[i] &= t.words[i]
	}
	s.words = s.words[:minLen]
}

func (s *IntSet) DifferenceWith(t *IntSet) {
	minLen := min(len(t.words), len(s.words))
	for i := range minLen {
		s.words[i] &^= t.words[i]
	}
}

func (s *IntSet) SymmetricDifference(t *IntSet) {
	minLen := min(len(t.words), len(s.words))
	for i := range minLen {
		s.words[i] ^= t.words[i]
	}
	if minLen == len(s.words) {
		s.words = append(s.words, t.words[minLen:]...)
	}
}

func (s *IntSet) Elems() []int {
	var elems []int
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < bitsPerWord; j++ {
			if word&(1<<uint(j)) != 0 {
				elems = append(elems, bitsPerWord*i+j)
			}
		}
	}
	return elems
}
