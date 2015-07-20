// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winapi_test

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/alexbrainman/winapi"
)

func TestGlobalMemoryStatusEx(t *testing.T) {
	var m winapi.MEMORYSTATUSEX
	m.Length = uint32(unsafe.Sizeof(m))
	err := winapi.GlobalMemoryStatusEx(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MEMORYSTATUSEX is %+v", m)
}

func TestGetProcessHandleCount(t *testing.T) {
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	var count uint32
	err = winapi.GetProcessHandleCount(h, &count)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Handle count is %v", count)
}

func TestGetVersionEx(t *testing.T) {
	var vi winapi.OSVERSIONINFOEX
	vi.OSVersionInfoSize = uint32(unsafe.Sizeof(vi))
	err := winapi.GetVersionEx(&vi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("OSVERSIONINFOEX is %+v", vi)
	t.Logf("OSVERSIONINFOEX.CSDVersion is %v", syscall.UTF16ToString(vi.CSDVersion[:]))
}
