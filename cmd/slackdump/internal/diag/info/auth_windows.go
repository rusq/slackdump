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

//go:build windows

package info

import (
	"context"
	"fmt"
	"io"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procLogonUserW = advapi32.NewProc("LogonUserW")
)

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

// Untested
func logonUser(username, domain, password string) (bool, error) {
	var token syscall.Handle
	r1, _, err := procLogonUserW.Call(
		uintptr(unsafe.Pointer(must(syscall.UTF16PtrFromString(username)))),
		uintptr(unsafe.Pointer(must(syscall.UTF16PtrFromString(domain)))),
		uintptr(unsafe.Pointer(must(syscall.UTF16PtrFromString(password)))),
		uintptr(2), // LOGON32_LOGON_INTERACTIVE
		uintptr(0), // LOGON32_PROVIDER_DEFAULT
		uintptr(unsafe.Pointer(&token)),
	)
	if r1 == 0 {
		return false, err
	}
	defer syscall.CloseHandle(token)
	return true, nil
}

func osValidateUser(_ context.Context, w io.Writer) error {
	fmt.Fprint(w, "Enter username: ")
	var username string
	fmt.Scanln(&username)
	fmt.Fprintf(w, "Enter password for %s: ", username)
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	domain := "." // Use "." for local account
	ok, err := logonUser(username, domain, string(password))
	if err != nil {
		return fmt.Errorf("authentication error: %w", err)
	}
	if !ok {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
