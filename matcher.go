package main

type Matcher struct {
	state   int
	pattern string
}

func NewMatcher(pattern string) *Matcher {
	return &Matcher{pattern: pattern}
}

func (m *Matcher) Match(buffer []byte) {
	state := m.state
	for i := 0; i < len(buffer) && state < len(m.pattern); i++ {
		if m.pattern[state] == buffer[i] {
			state++
		} else {
			state = 0
			if m.pattern[state] == buffer[i] {
				state++
			}
		}
	}
	m.state = state
}

func (m *Matcher) Matched() bool {
	return m.state == len(m.pattern)
}

func (m *Matcher) Reset() {
	m.state = 0
}
