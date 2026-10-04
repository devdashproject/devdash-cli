package commands

import (
	"bufio"
	"strings"
	"testing"
)

func TestConfirmDelete(t *testing.T) {
	cases := map[string]bool{"y\n": true, "yes\n": true, "Y\n": true, "n\n": false, "\n": false, "": false, "\x1b[A\n": false}
	for in, want := range cases {
		if got := confirmDelete(bufio.NewReader(strings.NewReader(in)), "subj", false); got != want {
			t.Errorf("confirmDelete(%q) = %v, want %v", in, got, want)
		}
	}
}
