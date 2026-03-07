// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build debug

package cfg

import (
	"flag"
	"os"
)

// Additional configuration variables for dev environment.
var (
	CPUProfile string
	MEMProfile string
)

func setDevFlags(fs *flag.FlagSet, mask FlagMask) {
	fs.StringVar(&CPUProfile, "cpuprofile", os.Getenv("CPU_PROFILE"), "write CPU profile to `file`")
	fs.StringVar(&MEMProfile, "memprofile", os.Getenv("MEM_PROFILE"), "write memory profile to `file`")
}
