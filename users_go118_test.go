//go:build go1.18
// +build go1.18

package slackdump

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzFilenameSplit(f *testing.F) {
	testInput := []string{
		"users.json",
		"channels.json",
	}
	for _, ti := range testInput {
		f.Add(ti)
	}
	f.Fuzz(func(t *testing.T, input string) {
		split := filenameSplit(input)
		joined := filenameJoin(split)
		assert.Equal(t, input, joined)
	})
}
