package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unsafe"
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
	var trans []*state
	if c == empty {
		trans = append(trans, st)
	}
	if nextStates, hasNext := m.transitions[st][c]; hasNext {
		trans = append(trans, nextStates...)
	}
	return trans
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

func (match match) convert() automata {
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

func (concat concat) convert() automata {
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

func concatAutomata(a, b automata) automata {
	a.transitions = mergeTransitions(a.transitions, b.transitions)
	a.transitions[a.accepting] = a.transitions[b.initial]
	delete(a.transitions, b.initial)
	a.accepting = b.accepting
	return a
}

// Match on possibility of 2 expressions
type union struct {
	expr1 *expr // Must not be nil
	expr2 *expr // May be nil, but must be filled in later
}

func (union union) convert() automata {
	newInitial := newState()
	newAccepting := newState()

	expr1Automata := (*union.expr1).convert()
	expr2automata := (*union.expr2).convert()
	transitions := mergeTransitions(expr1Automata.transitions, expr2automata.transitions)
	transitions[newInitial] = map[rune][]*state{
		empty: []*state{expr1Automata.initial, expr2automata.initial},
	}
	transitions[expr1Automata.accepting] = map[rune][]*state{
		empty: []*state{newAccepting},
	}
	transitions[expr2automata.accepting] = map[rune][]*state{
		empty: []*state{newAccepting},
	}

	return automata{
		initial:     newInitial,
		accepting:   newAccepting,
		transitions: transitions}
}

func mergeTransitions(a, b map[*state]map[rune][]*state) map[*state]map[rune][]*state {
	for state, transitions := range b {
		a[state] = transitions
	}
	return a
}

// Match on 0 or more occurrences of one expression
type kleene struct {
	expr expr
}

func (kleene kleene) convert() automata {
	newInitial := newState()
	newAccepting := newState()
	subAutomata := kleene.expr.convert()
	transitions := subAutomata.transitions
	transitions[newInitial] = map[rune][]*state{
		empty: []*state{subAutomata.initial, newAccepting},
	}
	transitions[subAutomata.accepting] = map[rune][]*state{
		empty: []*state{newAccepting},
	}

	return automata{
		initial:     newInitial,
		accepting:   newAccepting,
		transitions: transitions}
}

// ( ( ) )
// ()*
// a*
// (a|b)

func compile(reg string) (*automata, error) {
	var exprs []expr
	for _, c := range reg {
		var next expr
		switch c {
		case '*':
			if len(exprs) == 0 {
				return nil, errors.New("Invalid regular expression: expected expression before '*'")
			}
			next, exprs = exprs[len(exprs)-1], exprs[:len(exprs)-1]
			next = kleene{expr: next}
		default:
			next = match{c: c}
		}
		exprs = append(exprs, next)
	}
	automata := (concat{exprs: exprs}).convert()
	return &automata, nil
}

func printAutomata(automata *automata) {
	numToStateMap := make(map[int]*state)
	stateToNumMap := make(map[*state]int)
	for state := range automata.transitions {
		stateToNumMap[state] = int(uintptr(unsafe.Pointer(state)))
		numToStateMap[stateToNumMap[state]] = state
	}

	sortedNums := make([]int, len(numToStateMap))
	i := 0
	for num := range numToStateMap {
		sortedNums[i] = num
		i++
	}
	sort.Ints(sortedNums)

	stateToStandardNumMap := make(map[*state]int)
	for standard, num := range sortedNums {
		stateToStandardNumMap[numToStateMap[num]] = standard
	}

	fmt.Println("initial:", stateToStandardNumMap[automata.initial])
	fmt.Println("accepting:", stateToStandardNumMap[automata.accepting])
	fmt.Println("transitions:")
	for state, transitions := range automata.transitions {
		for c, states := range transitions {
			printableStates := make([]string, len(states))
			stateCounter := 0
			for _, nextState := range states {
				printableStates[stateCounter] = fmt.Sprintf("%d", stateToStandardNumMap[nextState])
				stateCounter++
			}
			fmt.Printf("%d on '%c' to %s\n", stateToStandardNumMap[state], c, strings.Join(printableStates, ", "))
		}
	}
}

func main() {
	if abcAutomata, err := compile("abc"); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("abc match:", matchAutomata(*abcAutomata, "abc"))
		fmt.Println("a match:", matchAutomata(*abcAutomata, "a"))
		fmt.Println("b match:", matchAutomata(*abcAutomata, "b"))
		fmt.Println("c match:", matchAutomata(*abcAutomata, "c"))
		fmt.Println("ac match:", matchAutomata(*abcAutomata, "ac"))
		fmt.Println("ab match:", matchAutomata(*abcAutomata, "ab"))
		fmt.Println("abcd match:", matchAutomata(*abcAutomata, "abcd"))
	}

	if kleeneAutomata, err := compile("a*bc"); err != nil {
		fmt.Println(err)
	} else {
		printAutomata(kleeneAutomata)

		fmt.Println("abc match:", matchAutomata(*kleeneAutomata, "abc"))
		fmt.Println("aabc match:", matchAutomata(*kleeneAutomata, "aabc"))
		fmt.Println("b match:", matchAutomata(*kleeneAutomata, "b"))
		fmt.Println("c match:", matchAutomata(*kleeneAutomata, "c"))
		fmt.Println("ac match:", matchAutomata(*kleeneAutomata, "ac"))
		fmt.Println("ab match:", matchAutomata(*kleeneAutomata, "ab"))
		fmt.Println("abcd match:", matchAutomata(*kleeneAutomata, "abcd"))
	}

	k, _ := compile("a*")
	printAutomata(k)
}
