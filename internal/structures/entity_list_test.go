package structures

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
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
				Include: []string{"one", "three", "two"},
			},
			false,
		},
		{
			"only excludes",
			args{[]string{"^one", "^two", "^three"}},
			&EntityList{
				Exclude: []string{"one", "three", "two"},
			},
			false,
		},
		{
			"mixed",
			args{[]string{"one", "^two", "three"}},
			&EntityList{
				Include: []string{"one", "three"},
				Exclude: []string{"two"},
			},
			false,
		},
		{
			"same element included and excluded",
			args{[]string{"one", "^two", "three", "two"}},
			&EntityList{
				Include: []string{"one", "three"},
				Exclude: []string{"two"},
			},
			false,
		},
		{
			"duplicate element",
			args{[]string{"one", "^two", "three", "one"}},
			&EntityList{
				Include: []string{"one", "three"},
				Exclude: []string{"two"},
			},
			false,
		},
		{
			"empty element",
			args{[]string{"one", "^two", "three", "", "four", "^"}},
			&EntityList{
				Include: []string{"four", "one", "three"},
				Exclude: []string{"two"},
			},
			false,
		},
		{
			"everything is empty",
			args{[]string{}},
			&EntityList{},
			false,
		},
		{
			"nil",
			args{nil},
			&EntityList{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeEntityList(tt.args.entities)
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

func Test_readEntityList(t *testing.T) {
	type args struct {
		r          io.Reader
		maxEntries int
	}
	tests := []struct {
		name    string
		args    args
		want    *EntityList
		wantErr bool
	}{
		{
			"two lines one comment, no CR",
			args{strings.NewReader("C555\n C123\n#C555\n^C321\n\t^C333"), maxFileEntries},
			&EntityList{
				Include: []string{"C123", "C555"},
				Exclude: []string{"C321", "C333"},
			},
			false,
		},
		{
			"two lines one comment",
			args{strings.NewReader("C123\n#C555\n^C321\n"), maxFileEntries},
			&EntityList{
				Include: []string{"C123"},
				Exclude: []string{"C321"},
			},
			false,
		},
		{
			"last line comment",
			args{strings.NewReader("C123\n #C555\n#C321\n"), maxFileEntries},
			&EntityList{Include: []string{"C123"}},
			false,
		},
		{
			"oneliner",
			args{strings.NewReader("C321"), maxFileEntries},
			&EntityList{Include: []string{"C321"}},
			false,
		},
		{
			"oneliner url",
			args{strings.NewReader("https://fake.slack.com/archives/CHM82GF99/p1577694990000400"), maxFileEntries},
			&EntityList{Include: []string{"CHM82GF99" + linkSep + "1577694990.000400"}},
			false,
		},
		{
			"excluded oneliner url",
			args{strings.NewReader("^https://fake.slack.com/archives/CHM82GF99/p1577694990000400"), maxFileEntries},
			&EntityList{Exclude: []string{"CHM82GF99" + linkSep + "1577694990.000400"}},
			false,
		},
		{
			"empty file",
			args{strings.NewReader(""), maxFileEntries},
			&EntityList{},
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
			got, err := readEntityList(tt.args.r, tt.args.maxEntries)
			if (err != nil) != tt.wantErr {
				t.Errorf("readEntityList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readEntityList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityList_Index(t *testing.T) {
	type fields struct {
		Include []string
		Exclude []string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]bool
	}{
		{
			"simple",
			fields{
				Include: []string{"C123", "C234"},
				Exclude: []string{"C456", "C567"},
			},
			map[string]bool{
				"C123": true,
				"C234": true,
				"C456": false,
				"C567": false,
			},
		},
		{
			"intersecting",
			fields{
				Include: []string{"C123", "C234"},
				Exclude: []string{"C234", "C567"},
			},
			map[string]bool{
				"C123": true,
				"C234": false,
				"C567": false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &EntityList{
				Include: tt.fields.Include,
				Exclude: tt.fields.Exclude,
			}
			if got := el.Index(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EntityList.Index() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityList_HasIncludes(t *testing.T) {
	type fields struct {
		Include []string
		Exclude []string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"yes",
			fields{Include: []string{"1"}},
			true,
		},
		{
			"no",
			fields{Include: []string{}},
			false,
		},
		{
			"no",
			fields{Exclude: []string{"2"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &EntityList{
				Include: tt.fields.Include,
				Exclude: tt.fields.Exclude,
			}
			if got := el.HasIncludes(); got != tt.want {
				t.Errorf("EntityList.HasIncludes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityList_HasExcludes(t *testing.T) {
	type fields struct {
		Include []string
		Exclude []string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"yes",
			fields{Exclude: []string{"1"}},
			true,
		},
		{
			"no",
			fields{Exclude: []string{}},
			false,
		},
		{
			"no",
			fields{Include: []string{"2"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &EntityList{
				Include: tt.fields.Include,
				Exclude: tt.fields.Exclude,
			}
			if got := el.HasExcludes(); got != tt.want {
				t.Errorf("EntityList.HasExcludes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityList_IsEmpty(t *testing.T) {
	type fields struct {
		Include []string
		Exclude []string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"empty",
			fields{},
			true,
		},
		{
			"not empty",
			fields{Include: []string{"1"}},
			false,
		},
		{
			"not empty",
			fields{Exclude: []string{"1"}},
			false,
		},
		{
			"not empty",
			fields{
				Include: []string{"1"},
				Exclude: []string{"2"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &EntityList{
				Include: tt.fields.Include,
				Exclude: tt.fields.Exclude,
			}
			if got := el.IsEmpty(); got != tt.want {
				t.Errorf("EntityList.IsEmpty() = %v, want %v", got, tt.want)
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
				"@" + mkTestFile(td, "OVERRIDE0ME\n^EXCLUDED\n#comment\nADD0ME"),
				"ANOTHER0INLINE0INCLUDE",
				"@" + mkTestFile(td, "SECOND0FILE0INCLUDE\n^SECOND0FILE0EXCLUDE"),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildEntityIndex(tt.args.entities)
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

func mkTestFile(dir string, content string) string {
	f, err := os.CreateTemp(dir, "")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := io.Copy(f, strings.NewReader(content)); err != nil {
		panic(err)
	}
	return f.Name()
}
