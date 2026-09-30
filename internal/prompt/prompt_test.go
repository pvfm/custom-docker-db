package prompt

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	for answer, want := range map[string]bool{
		"s\n": true, "SIM\n": true, "y\n": true, " Yes \n": true,
		"n\n": false, "\n": false, "": false, "talvez\n": false,
	} {
		var out bytes.Buffer
		if got := Confirm(strings.NewReader(answer), &out, "ok? "); got != want {
			t.Errorf("%q: got %v", answer, got)
		}
		if out.String() != "ok? " {
			t.Errorf("pergunta = %q", out.String())
		}
	}
}

func TestConfirmSharedReaderKeepsBufferedLines(t *testing.T) {
	in := bufio.NewReader(strings.NewReader("s\ns\n"))
	var out bytes.Buffer
	if !Confirm(in, &out, "1? ") || !Confirm(in, &out, "2? ") {
		t.Fatal("duas respostas em entrada canalizada deveriam ser lidas")
	}
}
