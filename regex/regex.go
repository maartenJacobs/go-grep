package regex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

// Automata represents the compiled regular expression. It can be used to test whether
// the regular expression describes a given string.
type Automata struct {
	initial     *state
	accepting   *state
	transitions map[*state]map[rune][]*state
}

func (m *Automata) getTransitions(st *state, c rune) []*state {
	var trans []*state
	if c == empty {
		trans = append(trans, st)
	}
	if nextStates, hasNext := m.transitions[st][c]; hasNext {
		trans = append(trans, nextStates...)
	}
	return trans
}

// Matches takes a string and checks if it can be described by the regular expression,
// expressed by the automata.
func (m Automata) Matches(in string) bool {
	matcher := newMatcher(m)
	for _, c := range in {
		matcher.move(c)
	}
	return matcher.isInAcceptingState()
}

type matcher struct {
	automata  Automata
	oldStates []*state
	newStates []*state
	alreadyOn map[*state]bool
}

func newMatcher(automata Automata) matcher {
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

func (matcher *matcher) isInAcceptingState() bool {
	for _, cstate := range matcher.oldStates {
		if matcher.automata.accepting == cstate {
			return true
		}
	}
	return false
}

type expr interface {
	convert() Automata
}

// Match on a single character
type match struct {
	c rune
}

func (match match) convert() Automata {
	state0 := newState()
	state1 := newState()

	transitions := map[*state]map[rune][]*state{
		state0: map[rune][]*state{match.c: []*state{state1}},
		state1: map[rune][]*state{},
	}

	return Automata{
		initial:     state0,
		accepting:   state1,
		transitions: transitions}
}

// Match on concatenation of multiple expressions in order
type concat struct {
	exprs []expr
}

func (concat concat) convert() Automata {
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

func concatAutomata(a, b Automata) Automata {
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

func (union union) convert() Automata {
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

	return Automata{
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

func (kleene kleene) convert() Automata {
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

	return Automata{
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

func (stack *exprStack) modifyLastExpr(modifier func(e expr) expr) {
	stack.push(modifier(stack.pop()))
}

func (stack exprStack) close() concat {
	return concat{exprs: stack.exprs}
}

func (stack *exprStack) closeUnion() {
	stack.exprs = []expr{
		union{expr1: stack.unionOption, expr2: stack.close()},
	}
	stack.unionOption = nil
}

func compileExpression(input *bufio.Reader, isClosed func(rune, error) bool) (expr, error) {
	stack := exprStack{}
	var (
		c   rune
		err error
	)

	for c, _, err = input.ReadRune(); !isClosed(c, err) && err == nil; c, _, err = input.ReadRune() {
		switch c {
		case '*':
			if len(stack.exprs) == 0 {
				return nil, errors.New("Invalid regular expression: expected expression before '*'")
			}
			stack.modifyLastExpr(func(e expr) expr { return kleene{expr: e} })

		case '(':
			expr, err := compileExpression(input, func(c rune, _ error) bool {
				return c == ')'
			})
			if err != nil {
				return nil, err
			}
			stack.push(expr)

		// The union operator '|' operates, as the name suggests, as the union of 2 options.
		// When found in a regular expression, it starts a new union expression with the current
		// stack of expressions as its first option.
		case '|':
			// Close the previous union.
			if stack.unionOption != nil {
				stack.closeUnion()
			}

			// Turn the current stack into the first union expression.
			stack.unionOption = stack.close()
			stack.exprs = []expr{}

		default:
			stack.push(match{c: c})
		}
	}
	if !isClosed(c, err) && err != nil {
		return nil, err
	}

	if stack.unionOption != nil {
		stack.closeUnion()
	}

	return stack.close(), nil
}

// Compile takes a regular expression as an input stream and returns an Automata
// as a result.
func Compile(input *bufio.Reader) (*Automata, error) {
	expr, err := compileExpression(input, func(_ rune, err error) bool {
		return err == io.EOF
	})
	if err != nil {
		return nil, err
	}
	automata := expr.convert()
	return &automata, nil
}

func printAutomata(automata *Automata) {
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
