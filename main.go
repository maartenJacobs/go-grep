package main

import (
	"fmt"
)

const empty rune = 0

type state uint8

func newState() *state {
	state := state(0)
	return &state
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
	automata  automata
	oldStates []*state
	newStates []*state
	alreadyOn map[*state]bool
}

func newMatcher(automata automata) matcher {
	matcher := matcher{
		automata:  automata,
		alreadyOn: make(map[*state]bool),
	}

	// Initialise alreadyOn. No state should be currently on the "new" states stack.
	for state := range matcher.automata.transitions {
		matcher.alreadyOn[state] = false
	}

	matcher.oldStates = append(matcher.oldStates, matcher.automata.initial)
	matcher.move(empty)
	return matcher
}

func (matcher *matcher) addState(state *state) {
	matcher.newStates = append(matcher.newStates, state)
	matcher.alreadyOn[state] = true
	for _, nextState := range matcher.automata.getTransitions(state, empty) {
		if !matcher.alreadyOn[nextState] {
			matcher.addState(nextState)
		}
	}
}

func (matcher *matcher) move(c rune) {
	for _, oldState := range matcher.oldStates {
		for _, nextState := range matcher.automata.getTransitions(oldState, c) {
			if !matcher.alreadyOn[nextState] {
				matcher.addState(nextState)
			}
		}
	}

	// Transfer new states to old states.
	matcher.oldStates = matcher.newStates
	for _, newState := range matcher.newStates {
		matcher.alreadyOn[newState] = false
	}
	matcher.newStates = make([]*state, 0)
}

func matchAutomata(m automata, in string) bool {
	matcher := newMatcher(m)
	for _, c := range in {
		matcher.move(c)
	}

	for _, cstate := range matcher.oldStates {
		if matcher.automata.accepting == cstate {
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

type expr interface {
	convert() automata
}

// Match on a single character
type match struct {
	c rune
}

func (match *match) convert() automata {
	state0 := newState()
	state1 := newState()

	transitions := map[*state]map[rune][]*state{
		state0: map[rune][]*state{match.c: []*state{state1}},
		state1: map[rune][]*state{},
	}

	return automata{
		initial:     state0,
		accepting:   state1,
		transitions: transitions}
}

// Match on concatenation of multiple expressions in order
type concat struct {
	exprs []expr
}

func (concat *concat) convert() automata {
	if len(concat.exprs) == 0 {
		panic("Concat expression has no subexpressions")
	}

	exprs := concat.exprs
	automata := exprs[0].convert()
	for _, expr := range exprs[1:] {
		automata = concatAutomata(automata, expr.convert())
	}
	return automata
}

// Match on possibility of 2 expressions
type union struct {
	expr1 *expr // Must not be nil
	expr2 *expr // May be nil, but must be filled in later
}

func (union *union) convert() automata {
	newInitial := newState()
	newAccepting := newState()

	expr1Automata := union.expr1.convert()
	expr2automata := union.expr2.convert()
	transitions := expr1Automata.transitions
	for k, v := range expr2Automata.transitions {
		transitions[k] = v
	}
}

func mergeTransitions(a, b map[*state]) {

}

// Match on 0 or more occurrences of one expression
type kleene struct {
	expr expr
}

func concatAutomata(a, b automata) automata {

}

func compile(reg string) automata {

}

func main() {
	m := buildStateMachine()
	fmt.Println("abb match:", matchAutomata(m, "abb"))
	fmt.Println("aabb match:", matchAutomata(m, "aabb"))
	fmt.Println("babb match:", matchAutomata(m, "babb"))
	fmt.Println("ab match:", matchAutomata(m, "ab"))
	fmt.Println("a match:", matchAutomata(m, "a"))
}
