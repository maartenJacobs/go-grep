package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	regex "github.com/maartenJacobs/go-grep/regex"
)

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

	automata, err := regex.Compile(bufio.NewReader(strings.NewReader(os.Args[1])))
	if err != nil {
		fmt.Println(err)
	} else {
		input := strings.TrimRight(line, "\n")
		fmt.Printf("Trying '%s' on '%s': %v\n", os.Args[1], input, automata.Matches(input))
	}
}
