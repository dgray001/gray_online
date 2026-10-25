package ai

type letState struct {
	specs       map[string]amount
	busy        map[string]bool
	cached      map[string]float64
	cache_depth int
	static      bool
}

func (s selItem) canMemoizeLet() bool {
	if s.lets.static {
		return true
	}
	return s.lets.cache_depth > 0 && s.lets.cache_depth == s.internals.binding_depth
}

func (s selItem) cachedLet(name string) (float64, bool) {
	if !s.canMemoizeLet() {
		return 0, false
	}
	value, found := s.lets.cached[name]
	return value, found
}

func (s selItem) cacheLet(name string, value float64, errors_before uint64) {
	if !s.canMemoizeLet() || s.internals.eval_errors != errors_before || s.internals.select_steps > maxSelectSteps {
		return
	}
	if s.lets.cached == nil {
		s.lets.cached = map[string]float64{}
	}
	s.lets.cached[name] = value
}
