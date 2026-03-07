package base

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrOpCancelled is returned when an operation is cancelled by the user.
var ErrOpCancelled = fmt.Errorf("operation cancelled")

func YesNo(message string) bool {
	return YesNoWR(os.Stdout, os.Stdin, message)
}

func YesNoWR(w io.Writer, r io.Reader, message string) bool {
	const pleaseAnswerYN = "Please answer yes or no and press Enter or Return."
	for {
		fmt.Fprint(w, message, "? (y/N) ")
		var resp string
		_, err := fmt.Fscanln(r, &resp)
		if err != nil {
			// there's no proper way to check for unexpected newline error.
			if strings.EqualFold(err.Error(), "unexpected newline") {
				return false
			}
			fmt.Fprintln(w, pleaseAnswerYN)
			continue
		}
		resp = strings.TrimSpace(resp)
		if len(resp) > 0 {
			switch strings.ToLower(resp)[0] {
			case 'y':
				return true
			case 'n':
				return false
			}
		}
		fmt.Fprintln(w, pleaseAnswerYN)
	}
}
