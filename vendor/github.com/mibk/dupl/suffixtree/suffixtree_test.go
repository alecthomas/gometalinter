package suffixtree

import "testing"

type char byte

func (c char) Val() int {
	return int(c)
}

func str2tok(str string) []Token {
	toks := make([]Token, len(str))
	for i, c := range str {
		toks[i] = char(c)
	}
	return toks
}

func TestConstruction(t *testing.T) {
	str := "cacao"
	_, s := genStates(8, str)
	// s[0] is root
	s[0].addTran(0, 1, s[1]) // ca
	s[0].addTran(1, 1, s[2]) // a
	s[0].addTran(4, 4, s[3]) // o

	s[1].addTran(2, 4, s[4]) // cao
	s[1].addTran(4, 4, s[5]) // o

	s[2].addTran(2, 4, s[4]) // cao
	s[2].addTran(4, 4, s[5]) // o

	cacao := New()
	cacao.Update(str2tok(str)...)
	compareTrees(t, s[0], cacao.root)

	str2 := "banana"
	_, r := genStates(4, str2)
	r[0].addTran(0, 5, r[1]) // banana
	r[0].addTran(1, 5, r[2]) // anana
	r[0].addTran(2, 5, r[3]) // nana

	banana := New()
	banana.Update(str2tok(str2)...)
	compareTrees(t, r[0], banana.root)

	_, q := genStates(11, str2+"$")
	// r[0] is root
	q[0].addTran(0, 6, q[1]) // banana$
	q[0].addTran(1, 1, q[2]) // a
	q[0].addTran(2, 3, q[3]) // na
	q[0].addTran(6, 6, q[4]) // $

	q[2].addTran(2, 3, q[5]) // na
	q[2].addTran(6, 6, q[6]) // $

	q[3].addTran(4, 6, q[7]) // na$
	q[3].addTran(6, 6, q[8]) // $

	q[5].addTran(4, 6, q[9])  // na$
	q[5].addTran(6, 6, q[10]) // $

	banana.Update(char('$'))
	compareTrees(t, q[0], banana.root)

	foo := New()
	foo.Update(str2tok("a b ac c ")...)
}

func compareTrees(t *testing.T, expected, actual *state) {
	ch1, ch2 := walker(expected), walker(actual)
	for {
		etran, ok1 := <-ch1
		atran, ok2 := <-ch2
		if !ok1 || !ok2 {
			if ok1 {
				t.Error("expected tree is longer")
			} else if ok2 {
				t.Error("actual tree is longer")
			}
			break
		}
		if etran.start != atran.start || etran.ActEnd() != atran.ActEnd() {
			t.Errorf("got transition (%d, %d) '%s', want (%d, %d) '%s'",
				atran.start, atran.ActEnd(), actual.tree.data[atran.start:atran.ActEnd()+1],
				etran.start, etran.ActEnd(), expected.tree.data[etran.start:etran.ActEnd()+1],
			)
		}
	}
}

func walker(s *state) <-chan *tran {
	ch := make(chan *tran)
	go func() {
		walk(s, ch)
		close(ch)
	}()
	return ch
}

func walk(s *state, ch chan<- *tran) {
	for _, tr := range s.trans {
		ch <- tr
		walk(tr.state, ch)
	}
}

func genStates(count int, data string) (*STree, []*state) {
	t := new(STree)
	t.data = str2tok(data)
	states := make([]*state, count)
	for i := range states {
		states[i] = newState(t)
	}
	return t, states
}

type refPair struct {
	s          *state
	start, end Pos
}

func TestCanonize(t *testing.T) {
	tree, s := genStates(5, "somebanana")
	tree.auxState, tree.root = s[4], s[0]
	s[0].addTran(0, 3, s[1])
	s[1].addTran(4, 6, s[2])
	s[2].addTran(7, infinity, s[3])

	find := func(needle *state) int {
		for i, state := range s {
			if state == needle {
				return i
			}
		}
		return -1
	}

	var testCases = []struct {
		origin, expected refPair
	}{
		{refPair{s[0], 0, 0}, refPair{s[0], 0, 0}},
		{refPair{s[0], 0, 2}, refPair{s[0], 0, 0}},
		{refPair{s[0], 0, 3}, refPair{s[1], 4, 0}},
		{refPair{s[0], 0, 8}, refPair{s[2], 7, 0}},
		{refPair{s[0], 0, 6}, refPair{s[2], 7, 0}},
		{refPair{s[0], 0, 100}, refPair{s[2], 7, 0}},
		{refPair{s[4], -1, 100}, refPair{s[2], 7, 0}},
	}

	for _, tc := range testCases {
		s, start := tree.canonize(tc.origin.s, tc.origin.start, tc.origin.end)
		if s != tc.expected.s || start != tc.expected.start {
			t.Errorf("for origin ref. pair (%d, (%d, %d)) got (%d, %d), want (%d, %d)",
				find(tc.origin.s), tc.origin.start, tc.origin.end,
				find(s), start,
				find(tc.expected.s), tc.expected.start,
			)
		}
	}
}

func TestSplitting(t *testing.T) {
	tree := new(STree)
	tree.data = str2tok("banana|cbao")
	s1 := newState(tree)
	s2 := newState(tree)
	s1.addTran(0, 3, s2)

	// active point is (s1, 0, -1), an explicit state
	tree.end = 7 // c
	rets, end := tree.testAndSplit(s1, 0, -1)
	if rets != s1 {
		t.Errorf("got state %p, want %p", rets, s1)
	}
	if end {
		t.Error("should not be an end-point")
	}
	tree.end = 8 // b
	_, end = tree.testAndSplit(s1, 0, -1)
	if !end {
		t.Error("should be an end-point")
	}

	// active point is (s1, 0, 2), an implicit state
	tree.end = 9 // a
	rets, end = tree.testAndSplit(s1, 0, 2)
	if rets != s1 {
		t.Error("returned state should be unchanged")
	}
	if !end {
		t.Error("should be an end-point")
	}

	// [s1]-banana->[s2] => [s1]-ban->[rets]-ana->[s2]
	tree.end = 10 // o
	rets, end = tree.testAndSplit(s1, 0, 2)
	tr := s1.findTran(char('b'))
	if tr == nil {
		t.Error("should have a b-transition")
	} else if tr.state != rets {
		t.Errorf("got state %p, want %p", tr.state, rets)
	}
	tr2 := rets.findTran(char('a'))
	if tr2 == nil {
		t.Error("should have an a-transition")
	} else if tr2.state != s2 {
		t.Errorf("got state %p, want %p", tr2.state, s2)
	}
	if end {
		t.Error("should not be an end-point")
	}
}

func TestPosMaxValue(t *testing.T) {
	var p Pos = infinity
	if p+1 > 0 {
		t.Error("const infinity is not max value")
	}
}

func BenchmarkConstruction(b *testing.B) {
	stream := str2tok(`all work and no play makes jack a dull boy
all work and no play makes jack a dull boy
all work and no play makes jack a dull boy`)

	for i := 0; i < b.N; i++ {
		t := New()
		t.Update(stream...)
	}
}
