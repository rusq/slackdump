package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	got := All()

	assert.Equal(t, Types{CCSV, CJSON, CText}, got)
}
