package cfg

import "strings"

const zipExt = ".ZIP"

// StripZipExt removes the .zip extension from the string.
func StripZipExt(s string) string {
	if strings.HasSuffix(strings.ToUpper(s), zipExt) {
		return s[:len(s)-len(zipExt)]
	}
	return s
}
