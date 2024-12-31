package structures

import (
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestHasExcludePrefix(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"has exclude prefix",
			args{"^not this"},
			true,
		},
		{
			"this can't be happening",
			args{"t^his"},
			false,
		},
		{
			"definitely no",
			args{"this"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasExcludePrefix(tt.args.s); got != tt.want {
				t.Errorf("HasExcludePrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeEntityList(t *testing.T) {
	type args struct {
		entities []string
	}
	tests := []struct {
		name    string
		args    args
		want    *EntityList
		wantErr bool
	}{
		{
			"only includes",
			args{[]string{"one", "two", "three"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: true,
					},
					"three": {
						Id:      "three",
						Include: true,
					},
					"two": {
						Id:      "two",
						Include: true,
					},
				},
				hasIncludes: true,
				hasExcludes: false,
			},
			false,
		},
		{
			"only excludes",
			args{[]string{"^one", "^two", "^three"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: false,
					},
					"three": {
						Id:      "three",
						Include: false,
					},
					"two": {
						Id:      "two",
						Include: false,
					},
				},
				hasIncludes: false,
				hasExcludes: true,
			},
			false,
		},
		{
			"mixed",
			args{[]string{"one", "^two", "three"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: true,
					},
					"three": {
						Id:      "three",
						Include: true,
					},
					"two": {
						Id:      "two",
						Include: false,
					},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"same element included and excluded",
			args{[]string{"one", "^two", "three", "two"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: true,
					},
					"three": {
						Id:      "three",
						Include: true,
					},
					"two": {
						Id:      "two",
						Include: false,
					},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"duplicate element",
			args{[]string{"one", "^two", "three", "one"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: true,
					},
					"three": {
						Id:      "three",
						Include: true,
					},
					"two": {
						Id:      "two",
						Include: false,
					},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"empty element",
			args{[]string{"one", "^two", "three", "", "four", "^"}},
			&EntityList{
				index: map[string]*EntityItem{
					"four": {
						Id:      "four",
						Include: true,
					},
					"one": {
						Id:      "one",
						Include: true,
					},
					"three": {
						Id:      "three",
						Include: true,
					},
					"two": {
						Id:      "two",
						Include: false,
					},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"everything is empty",
			args{[]string{}},
			&EntityList{
				index: map[string]*EntityItem{},
			},
			false,
		},
		{
			"nil",
			args{nil},
			&EntityList{
				index: map[string]*EntityItem{},
			},
			false,
		},
		{
			"with date limits",
			args{[]string{"one,,", "^two,,", "three,bad,2024-01-10T23:02:12", "four,2023-12-10T23:02:12,2024-01-10T23:02:12", "^five,2023-12-10T23:02:12,2024-01-10T23:02:12", "six,2023-12-10T23:02:12,2024-01-10T23:02:12,,", "seven,2024-04-07T23:02:12"}},
			&EntityList{
				index: map[string]*EntityItem{
					"one": {
						Id:      "one",
						Include: true,
						Oldest:  time.Time{},
						Latest:  time.Time{},
					},
					"two": {
						Id:      "two",
						Include: false,
						Oldest:  time.Time{},
						Latest:  time.Time{},
					},
					"three": {
						Id:      "three",
						Include: true,
						Oldest:  time.Time{},
						Latest:  time.Date(2024, time.January, 10, 23, 2, 12, 0, time.UTC),
					},
					"four": {
						Id:      "four",
						Include: true,
						Oldest:  time.Date(2023, time.December, 10, 23, 2, 12, 0, time.UTC),
						Latest:  time.Date(2024, time.January, 10, 23, 2, 12, 0, time.UTC),
					},
					"five": {
						Id:      "five",
						Include: false,
						Oldest:  time.Date(2023, time.December, 10, 23, 2, 12, 0, time.UTC),
						Latest:  time.Date(2024, time.January, 10, 23, 2, 12, 0, time.UTC),
					},
					"six": {
						Id:      "six",
						Include: true,
						Oldest:  time.Date(2023, time.December, 10, 23, 2, 12, 0, time.UTC),
						Latest:  time.Time{},
					},
					"seven": {
						Id:      "seven",
						Include: true,
						Oldest:  time.Date(2024, time.April, 7, 23, 2, 12, 0, time.UTC),
						Latest:  time.Time{},
					},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEntityList(tt.args.entities)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeEntityList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeEntityList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readEntityIndex(t *testing.T) {
	type args struct {
		r          io.Reader
		maxEntries int
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{
			"two lines one comment, no CR",
			args{strings.NewReader("C555\n C123\n#C555\n^C321\n\t^C333"), maxFileEntries},
			map[string]bool{
				"C123": true,
				"C555": true,
				"C321": false,
				"C333": false,
			},
			false,
		},
		{
			"two lines one comment",
			args{strings.NewReader("C123\n#C555\n^C321\n"), maxFileEntries},
			map[string]bool{
				"C123": true,
				"C321": false,
			},
			false,
		},
		{
			"last line comment",
			args{strings.NewReader("C123\n #C555\n#C321\n"), maxFileEntries},
			map[string]bool{
				"C123": true,
			},
			false,
		},
		{
			"oneliner",
			args{strings.NewReader("C321"), maxFileEntries},
			map[string]bool{
				"C321": true,
			},
			false,
		},
		{
			"oneliner url",
			args{strings.NewReader("https://fake.slack.com/archives/CHM82GF99/p1577694990000400"), maxFileEntries},
			map[string]bool{
				"CHM82GF99" + LinkSep + "1577694990.000400": true,
			},
			false,
		},
		{
			"excluded oneliner url",
			args{strings.NewReader("^https://fake.slack.com/archives/CHM82GF99/p1577694990000400"), maxFileEntries},
			map[string]bool{
				"CHM82GF99" + LinkSep + "1577694990.000400": false,
			},
			false,
		},
		{
			"empty file",
			args{strings.NewReader(""), maxFileEntries},
			map[string]bool{},
			false,
		},
		{
			"exceeding number of lines",
			args{strings.NewReader("ONE\nTWO\nTHREE\nFOUR"), 3},
			nil,
			true,
		},
		{
			"invalid URL",
			args{strings.NewReader("https://lol.co\n"), maxFileEntries},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readEntityIndex(tt.args.r, tt.args.maxEntries)
			if (err != nil) != tt.wantErr {
				t.Errorf("readEntityIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readEntityIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityList_Index(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want EntityIndex
	}{
		{
			"simple",
			[]string{"C123", "C234", "^C456", "^C567"},
			EntityIndex{
				"C123": true,
				"C234": true,
				"C456": false,
				"C567": false,
			},
		},
		{
			"intersecting",
			[]string{"C123", "C234", "^C234", "^C567"},
			EntityIndex{
				"C123": true,
				"C234": false,
				"C567": false,
			},
		},
		{
			"nil",
			nil,
			EntityIndex{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if el, err := NewEntityList(tt.args); err == nil {
				index := el.Index()
				for k, include := range tt.want {
					if item, ok := index[k]; !ok || item.Include != include {
						t.Errorf("EntityList.Index()[%v] = %v", k, include)
					}
				}
			} else {
				t.Errorf("EntityList.Index() = %v; error: %v", tt.want, err)
			}
		})
	}
}

func TestEntityList_HasIncludes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			"yes",
			[]string{"A1"},
			true,
		},
		{
			"no",
			[]string{},
			false,
		},
		{
			"no",
			[]string{"^A2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if el, err := NewEntityList(tt.args); err == nil {
				if got := el.HasIncludes(); got != tt.want {
					t.Errorf("EntityList.HasIncludes() = %v, want %v", got, tt.want)
				}
			} else {
				t.Errorf("EntityList.HasIncludes() = %v; error: %v", tt.want, err)
			}
		})
	}
}

func TestEntityList_HasExcludes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			"yes",
			[]string{"^A1"},
			true,
		},
		{
			"no",
			[]string{},
			false,
		},
		{
			"no",
			[]string{"A2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if el, err := NewEntityList(tt.args); err == nil {
				if got := el.HasExcludes(); got != tt.want {
					t.Errorf("EntityList.HasExcludes() = %v, want %v", got, tt.want)
				}
			} else {
				t.Errorf("EntityList.HasExcludes() = %v; error: %v", tt.want, err)
			}
		})
	}
}

func TestEntityList_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			"empty",
			[]string{},
			true,
		},
		{
			"not empty",
			[]string{"A1"},
			false,
		},
		{
			"not empty",
			[]string{"^A1"},
			false,
		},
		{
			"not empty",
			[]string{"A1", "^A2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if el, err := NewEntityList(tt.args); err == nil {
				if got := el.IsEmpty(); got != tt.want {
					t.Errorf("EntityList.IsEmpty() = %v, want %v", got, tt.want)
				}
			} else {
				t.Errorf("EntityList.IsEmpty() = %v; error: %v", tt.want, err)
			}
		})
	}
}

func Test_buildEntityIndex(t *testing.T) {
	td := t.TempDir()
	type args struct {
		entities []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{
			"ok",
			args{[]string{"C123", "C234", "^C345", "C456"}},
			map[string]bool{
				"C123": true,
				"C234": true,
				"C345": false,
				"C456": true,
			},
			false,
		},
		{
			"make sure excluded items don't get included later",
			args{[]string{"C123", "C234", "^C345", "C345", "C456"}},
			map[string]bool{
				"C123": true,
				"C234": true,
				"C345": false,
				"C456": true,
			},
			false,
		},
		{
			"file logic override",
			args{[]string{
				"^OVERRIDE0ME",
				"INLINE0INCLUDE",
				"^INLINE0EXCLUDE",
				"@" + fixtures.MkTestFile(t, td, "OVERRIDE0ME\n^EXCLUDED\n#comment\nADD0ME"),
				"ANOTHER0INLINE0INCLUDE",
				"@" + fixtures.MkTestFile(t, td, "SECOND0FILE0INCLUDE\n^SECOND0FILE0EXCLUDE"),
			}},
			map[string]bool{
				"OVERRIDE0ME":            false,
				"INLINE0INCLUDE":         true,
				"INLINE0EXCLUDE":         false,
				"EXCLUDED":               false,
				"ADD0ME":                 true,
				"ANOTHER0INLINE0INCLUDE": true,
				"SECOND0FILE0INCLUDE":    true,
				"SECOND0FILE0EXCLUDE":    false,
			},
			false,
		},
		{
			"with dates",
			args{[]string{
				"one,,",
				"^two,,",
				"three,bad,2024-01-10T23:02:12",
				"four,2023-12-10T23:02:12,2024-01-10T23:02:12",
				"^five,2023-12-10T23:02:12,2024-01-10T23:02:12",
				"six,2023-12-10T23:02:12,2024-01-10T23:02:12,,",
			}},
			map[string]bool{
				"one,,":                         true,
				"two,,":                         false,
				"three,bad,2024-01-10T23:02:12": true,
				"four,2023-12-10T23:02:12,2024-01-10T23:02:12":  true,
				"five,2023-12-10T23:02:12,2024-01-10T23:02:12":  false,
				"six,2023-12-10T23:02:12,2024-01-10T23:02:12,,": true,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildEntryIndex(tt.args.entities)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildEntityIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildEntityIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEntityList(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"ok",
			args{"C123 C234 ^C345 C456"},
			false,
		},
		{
			"empty",
			args{""},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateEntityList(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntityList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEntityListFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *EntityList
		wantErr bool
	}{
		{
			"ok",
			args{"C123 C234 ^C345 C456"},
			&EntityList{
				index: map[string]*EntityItem{
					"C123": {Id: "C123", Include: true},
					"C234": {Id: "C234", Include: true},
					"C345": {Id: "C345", Include: false},
					"C456": {Id: "C456", Include: true},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"empty",
			args{""},
			nil,
			true,
		},
		{
			"only includes",
			args{"one two three"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: true},
					"three": {Id: "three", Include: true},
					"two":   {Id: "two", Include: true},
				},
				hasIncludes: true,
				hasExcludes: false,
			},
			false,
		},
		{
			"only excludes",
			args{"^one ^two ^three"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: false},
					"two":   {Id: "two", Include: false},
					"three": {Id: "three", Include: false},
				},
				hasIncludes: false,
				hasExcludes: true,
			},
			false,
		},
		{
			"mixed",
			args{"one ^two three"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: true},
					"two":   {Id: "two", Include: false},
					"three": {Id: "three", Include: true},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"same element included and excluded",
			args{"one ^two three two"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: true},
					"two":   {Id: "two", Include: false},
					"three": {Id: "three", Include: true},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"duplicate element",
			args{"one ^two three one"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: true},
					"two":   {Id: "two", Include: false},
					"three": {Id: "three", Include: true},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"empty element",
			args{"one ^two three  four ^"},
			&EntityList{
				index: map[string]*EntityItem{
					"one":   {Id: "one", Include: true},
					"two":   {Id: "two", Include: false},
					"three": {Id: "three", Include: true},
					"four":  {Id: "four", Include: true},
				},
				hasIncludes: true,
				hasExcludes: true,
			},
			false,
		},
		{
			"everything is empty",
			args{""},
			nil,
			true,
		},
		{
			"nil",
			args{""},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEntityListFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEntityListFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEntityListFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
