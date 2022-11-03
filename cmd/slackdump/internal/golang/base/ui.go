package base

import (
	"fmt"
	"strings"
)

func YesNo(message string) bool {
	for {
		fmt.Print(message, "? (y/N) ")
		var resp string
		fmt.Scanln(&resp)
		resp = strings.TrimSpace(resp)
		if len(resp) > 0 {
			switch strings.ToLower(resp)[0] {
			case 'y':
				return true
			case 'n':
				return false
			}
		}
		fmt.Println("Please answer yes or no and press Enter or Return.")
	}
}
