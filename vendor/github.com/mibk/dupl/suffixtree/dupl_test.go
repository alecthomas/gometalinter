package suffixtree

import (
	"fmt"
	"sort"
	"testing"
)

func (m Match) String() string {
	str := "(["
	for _, p := range m.Ps {
		str += fmt.Sprintf("%d, ", p)
	}
	return str[:len(str)-2] + fmt.Sprintf("], %d)", m.Len)
}

func sliceCmp(sl1, sl2 []Pos) bool {
	if len(sl1) != len(sl2) {
		return false
	}
	sort.Sort(ByPos(sl1))
	sort.Sort(ByPos(sl2))
	for i := range sl1 {
		if sl1[i] != sl2[i] {
			return false
		}
	}
	return true
}

type ByPos []Pos

func (p ByPos) Len() int {
	return len(p)
}

func (p ByPos) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ByPos) Less(i, j int) bool {
	return p[i] < p[j]
}

func TestFindingDupl(t *testing.T) {
	testCases := []struct {
		s         string
		threshold int
		matches   []Match
	}{
		{"abab$", 3, []Match{}},
		{"abab$", 2, []Match{{[]Pos{0, 2}, 2}}},
		{"abcbcabc$", 3, []Match{{[]Pos{0, 5}, 3}}},
		{"abcbcabc$", 2, []Match{{[]Pos{0, 5}, 3}, {[]Pos{1, 3, 6}, 2}}},
		{`All work and no play makes Jack a dull boy
All work and no play makes Jack a dull boy$`, 4, []Match{{[]Pos{0, 43}, 42}}},
	}

	for _, tc := range testCases {
		tree := New()
		tree.Update(str2tok(tc.s)...)
		ch := tree.FindDuplOver(tc.threshold)
		for _, exp := range tc.matches {
			act, ok := <-ch
			if !ok {
				t.Errorf("missing match %v for '%s'", exp, tc.s)
			} else if exp.Len != act.Len || !sliceCmp(exp.Ps, act.Ps) {
				t.Errorf("got %v, want %v", act, exp)
			}
		}
		for act := range ch {
			t.Errorf("beyond expected match %v for '%s'", act, tc.s)
		}
	}
}
