package export

import (
	"context"
	"encoding/json"
	"os"
	"runtime/trace"
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/fixtures/fixgen"
	"github.com/rusq/slackdump/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversation_ByDate(t *testing.T) {
	var exp Export

	conversations := fixtures.Load[types.Conversation](fixtures.TestConversationJSON)
	users := fixtures.Load[types.Users](fixtures.UsersJSON)

	convDt, err := exp.byDate(&conversations, users.IndexByID())
	if err != nil {
		t.Fatal(err)
	}

	// uncomment to write the json for fixtures
	require.NoError(t, writeOutput("convDt", convDt))

	want := fixtures.Load[messagesByDate](fixtures.TestConversationExportJSON)

	// we need to depopulate slackdumpTime for comparison, as it is not saved
	// in the fixture.
	zeroSlackdumpTime(convDt)
	assert.Equal(t, want, convDt)
}

func zeroSlackdumpTime(m messagesByDate) {
	for _, msgs := range m {
		for i := range msgs {
			msgs[i].slackdumpTime = time.Time{}
		}
	}
}

func writeOutput(name string, v interface{}) error {
	f, err := os.Create(name + ".json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(v)
}

func Test_messagesByDate_validate(t *testing.T) {
	tests := []struct {
		name    string
		mbd     messagesByDate
		wantErr bool
	}{
		{"valid",
			messagesByDate{
				"2019-09-16": []*ExportMessage{},
				"2020-12-31": []*ExportMessage{},
			},
			false,
		},
		{"empty key",
			messagesByDate{
				"":           []*ExportMessage{},
				"2020-12-31": []*ExportMessage{},
			},
			true,
		},
		{"invalid key",
			messagesByDate{
				"2019-09-16": []*ExportMessage{},
				"2020-31-12": []*ExportMessage{}, //swapped month and date
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mbd.validate(); (err != nil) != tt.wantErr {
				t.Errorf("messagesByDate.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var (
	benchResult messagesByDate
	benchConv   types.Conversation
)

func init() {
	var (
		startDate   = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate     = time.Date(2020, 1, 1, 15, 0, 0, 0, time.UTC)
		numMessages = 10_000
	)
	benchConv = fixgen.GenerateTestConversation("test", startDate, endDate, numMessages)
}

func BenchmarkByDate(b *testing.B) {

	ctx, task := trace.NewTask(context.Background(), "BenchmarkByDate")
	defer task.End()

	var (
		ex  Export
		err error
	)
	region := trace.StartRegion(ctx, "byDateBenchRun")
	defer region.End()
	var m messagesByDate
	for i := 0; i < b.N; i++ {
		m, err = ex.byDate(&benchConv, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	benchResult = m
}
