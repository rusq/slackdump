package base

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func YesNo(message string) bool {
	return YesNoWR(os.Stdout, os.Stdin, message)
}

func YesNoWR(w io.Writer, r io.Reader, message string) bool {
	for {
		fmt.Fprint(w, message, "? (y/N) ")
		var resp string
		fmt.Fscanln(r, &resp)
		resp = strings.TrimSpace(resp)
		if len(resp) > 0 {
			switch strings.ToLower(resp)[0] {
			case 'y':
				return true
			case 'n':
				return false
			}
		}
		fmt.Fprintln(w, "Please answer yes or no and press Enter or Return.")
	}
}
