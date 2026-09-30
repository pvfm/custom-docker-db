// Package prompt asks yes/no questions on a terminal.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Confirm writes the question and reads one line from in. Only s, sim, y and
// yes (any case) mean yes; anything else, including EOF, means no.
//
// Pass the same *bufio.Reader to every Confirm call of a command: a fresh
// reader per call would drop lines already buffered from piped input.
func Confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprint(out, question)
	answer, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "s", "sim", "y", "yes":
		return true
	}
	return false
}
