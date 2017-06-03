package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
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
	expr1 expr
	expr2 expr
}

func (union union) convert() automata {
	newInitial := newState()
	newAccepting := newState()

	expr1Automata := union.expr1.convert()
	expr2Automata := union.expr2.convert()
	transitions := mergeTransitions(expr1Automata.transitions, expr2Automata.transitions)
	transitions[newInitial] = map[rune][]*state{
		empty: []*state{expr1Automata.initial, expr2Automata.initial},
	}
	transitions[expr1Automata.accepting] = map[rune][]*state{
		empty: []*state{newAccepting},
	}
	transitions[expr2Automata.accepting] = map[rune][]*state{
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
		empty: []*state{newAccepting, subAutomata.initial},
	}
	transitions[newAccepting] = map[rune][]*state{}

	return automata{
		initial:     newInitial,
		accepting:   newAccepting,
		transitions: transitions}
}

// The `exprStack` is an order collection of expressions that are converted to a single expression.
type exprStack struct {
	exprs       []expr
	unionOption expr
}

func newExprStack() exprStack {
	return exprStack{exprs: make([]expr, 0)}
}

func (stack *exprStack) push(ex expr) {
	stack.exprs = append(stack.exprs, ex)
}

func (stack *exprStack) pop() expr {
	var p expr
	p, stack.exprs = stack.exprs[len(stack.exprs)-1], stack.exprs[:len(stack.exprs)-1]
	return p
}

func (stack exprStack) close() concat {
	return concat{exprs: stack.exprs}
}

func compile(reg string) (*automata, error) {
	// Initialise the stack of expression stacks with the first stack to represent the top level expression.
	exprStacks := []exprStack{newExprStack()}
	expectedClosing := 0
	var next expr
	var currStack *exprStack
	for _, c := range reg {
		fmt.Println("Processing", string(c))
		currStack = &exprStacks[len(exprStacks)-1]

		switch c {
		case '*':
			if len(currStack.exprs) == 0 {
				return nil, errors.New("Invalid regular expression: expected expression before '*'")
			}
			next = kleene{expr: currStack.pop()}
		case '(':
			expectedClosing++
			exprStacks = append(exprStacks, newExprStack())
		case ')':
			if expectedClosing == 0 {
				return nil, errors.New("Unexpected closing parenthesis")
			}
			expectedClosing--

			// Close the current stack and append it to the previous stack.
			exprStacks = exprStacks[:len(exprStacks)-1]
			close := currStack.close()
			exprStacks[len(exprStacks)-1].push(close)

		// The union operator '|' operates, as the name suggests, as the union of 2 options.
		// When found in a regular expression, it starts a new union expression with the current
		// stack of expressions as its first option.
		case '|':
			// The union operator starts a new stack, so we need to complete the previous union.
			// if unionOption != nil {
			// 	exprs, stackStarts = closeUnion(unionOption, exprs, stackStarts)
			// }

			// Turn the current stack into the first union expression.
			currStack.unionOption = currStack.close()
			currStack.exprs = []expr{}

			// Start a new stack.
			exprStacks = append(exprStacks, newExprStack())
		default:
			next = match{c: c}
		}

		fmt.Println("next", next)
		if next != nil {
			fmt.Println("curr stack before push", currStack)
			currStack.push(next)
			next = nil
		}

		fmt.Println("curr stack", currStack)
		fmt.Println("expr stacks", exprStacks)
	}

	if expectedClosing != 0 {
		return nil, errors.New("Expected closing parenthesis")
	}

	if len(exprStacks) == 2 && exprStacks[0].unionOption != nil {
		lastUnion := union{expr1: exprStacks[0].unionOption, expr2: exprStacks[1].close()}
		exprStacks = exprStacks[:1]
		exprStacks[0].push(lastUnion)
	}

	if len(exprStacks) > 1 {
		// TODO: indicates a bug?
		fmt.Println("count of expr stacks unexpected:", exprStacks)
	}

	automata := exprStacks[0].close().convert()
	return &automata, nil
}

func printAutomata(automata *automata) {
	numToStateMap := make(map[int]*state)
	for state, transitions := range automata.transitions {
		numToStateMap[int(uintptr(unsafe.Pointer(state)))] = state
		for _, nextStates := range transitions {
			for _, nextState := range nextStates {
				nextStateNum := int(uintptr(unsafe.Pointer(nextState)))
				if _, nextStateSet := numToStateMap[nextStateNum]; !nextStateSet {
					numToStateMap[nextStateNum] = nextState
				}
			}
		}
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
		fmt.Print(stateToStandardNumMap[state])
		if len(transitions) == 0 {
			fmt.Println()
		}
		for c, states := range transitions {
			printableStates := make([]string, len(states))
			stateCounter := 0
			for _, nextState := range states {
				printableStates[stateCounter] = fmt.Sprintf("%d", stateToStandardNumMap[nextState])
				stateCounter++
			}
			fmt.Printf("\t'%c' -> %s\n", c, strings.Join(printableStates, ", "))
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-grep expr")
		os.Exit(2)
	}

	stdin := bufio.NewReader(os.Stdin)
	line, err := stdin.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	automata, err := compile(os.Args[1])
	if err != nil {
		fmt.Println(err)
	} else {
		printAutomata(automata)
		fmt.Println(matchAutomata(*automata, strings.TrimRight(line, "\n")))
	}
}
