// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package bootstrap

import (
	"reflect"
	"testing"
)

func Test_redactDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "separate value",
			args: []string{"archive", "-database-url", "postgres://user:secret@db/archive", "-o", "/data"},
			want: []string{"archive", "-database-url", "<redacted>", "-o", "/data"},
		},
		{
			name: "single dash assignment",
			args: []string{"resume", "-database-url=postgres://user:secret@db/archive", "/data"},
			want: []string{"resume", "-database-url=<redacted>", "/data"},
		},
		{
			name: "double dash assignment",
			args: []string{"resume", "--database-url=postgres://user:secret@db/archive", "/data"},
			want: []string{"resume", "--database-url=<redacted>", "/data"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactDatabaseURL(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("redactDatabaseURL() = %#v; want %#v", got, tt.want)
			}
		})
	}
}
