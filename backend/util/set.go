package util

// copied from https://gist.github.com/bgadrian/cb8b9344d9c66571ef331a14eb7a2e80

type Set struct {
	list map[int]struct{} //empty structs occupy 0 memory
}

func (s *Set) Has(v int) bool {
	_, ok := s.list[v]
	return ok
}

func (s *Set) Add(v int) {
	s.list[v] = struct{}{}
}

func (s *Set) Remove(v int) {
	delete(s.list, v)
}

func (s *Set) Clear() {
	s.list = make(map[int]struct{})
}

func (s *Set) Size() int {
	return len(s.list)
}

func NewSet() *Set {
	s := &Set{}
	s.list = make(map[int]struct{})
	return s
}

//optional functionalities

// AddMulti Add multiple values in the set
func (s *Set) AddMulti(list ...int) {
	for _, v := range list {
		s.Add(v)
	}
}

type FilterFunc func(v int) bool

// Filter returns a subset, that contains only the values that satisfies the given predicate P
func (s *Set) Filter(P FilterFunc) *Set {
	res := NewSet()
	for v := range s.list {
		if P(v) == false {
			continue
		}
		res.Add(v)
	}
	return res
}

func (s *Set) Union(s2 *Set) *Set {
	res := NewSet()
	for v := range s.list {
		res.Add(v)
	}

	for v := range s2.list {
		res.Add(v)
	}
	return res
}

func (s *Set) Intersect(s2 *Set) *Set {
	res := NewSet()
	for v := range s.list {
		if s2.Has(v) == false {
			continue
		}
		res.Add(v)
	}
	return res
}

// Difference returns the subset from s, that doesn't exists in s2 (param)
func (s *Set) Difference(s2 *Set) *Set {
	res := NewSet()
	for v := range s.list {
		if s2.Has(v) {
			continue
		}
		res.Add(v)
	}
	return res
}
