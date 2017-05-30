package main

import (
	"fmt"
)

const empty rune = 0

type state uint8

func newState() (state *state) {
	return
}

type automata struct {
	initial     *state
	accepting   *state
	transitions map[*state]map[rune][]*state
}

func (m *automata) getTransitions(st *state, c rune) []*state {
	if nextStates, hasNext := m.transitions[st][c]; hasNext {
		return nextStates
	}
	return make([]*state, 0)
}

type matcher struct {
	m         automata
	oldStates []int
	newStates []int
	alreadyOn []bool
}

func newMatcher(m automata) matcher {
	matcher := matcher{
		m:         m,
		alreadyOn: make([]bool, len(m.states)),
	}

	matcher.oldStates = append(matcher.oldStates, 0)
	matcher.move(empty)
	return matcher
}

func (m *matcher) addState(state int) {
	m.newStates = append(m.newStates, state)
	m.alreadyOn[state] = true
	for _, nextState := range m.m.getTransitions(state, empty) {
		if !m.alreadyOn[nextState] {
			m.addState(nextState)
		}
	}
}

func (m *matcher) move(c rune) {
	for _, oldState := range m.oldStates {
		for _, nextState := range m.m.getTransitions(oldState, c) {
			if !m.alreadyOn[nextState] {
				m.addState(nextState)
			}
		}
	}

	// Transfer new states to old states.
	m.oldStates = m.newStates
	for _, newState := range m.newStates {
		m.alreadyOn[newState] = false
	}
	m.newStates = make([]int, 0)
}

func main() {
	m := buildStateMachine()
	fmt.Println("abb match:", match(m, "abb"))
	fmt.Println("aabb match:", match(m, "aabb"))
	fmt.Println("babb match:", match(m, "babb"))
	fmt.Println("ab match:", match(m, "ab"))
	fmt.Println("a match:", match(m, "a"))
}

func match(m automata, in string) bool {
	matcher := newMatcher(m)
	for _, c := range in {
		matcher.move(c)
	}

	fmt.Println(matcher.oldStates)
	for _, cstate := range matcher.oldStates {
		if matcher.m.states[cstate].accepting {
			return true
		}
	}
	return false
}

func buildStateMachine() automata {
	state0 := newState()
	state1 := newState()
	state2 := newState()
	state3 := newState()
	state4 := newState()
	state5 := newState()
	state6 := newState()
	state7 := newState()
	state8 := newState()
	state9 := newState()
	state10 := newState()
	transitions := map[*state]map[rune][]*state{
		state0:  map[rune][]*state{empty: []*state{state1, state7}},
		state1:  map[rune][]*state{empty: []*state{state2, state4}},
		state2:  map[rune][]*state{'a': []*state{state3}},
		state3:  map[rune][]*state{empty: []*state{state6}},
		state4:  map[rune][]*state{'b': []*state{state5}},
		state5:  map[rune][]*state{empty: []*state{state6}},
		state6:  map[rune][]*state{empty: []*state{state1, state7}},
		state7:  map[rune][]*state{'a': []*state{state8}},
		state8:  map[rune][]*state{'b': []*state{state9}},
		state9:  map[rune][]*state{'b': []*state{state10}},
		state10: map[rune][]*state{},
	}

	return automata{
		initial:     state0,
		accepting:   state10,
		transitions: transitions}
}

// func compile(reg string) stateMachine {

// }
