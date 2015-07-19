// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winapi_test

import (
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
	t.Logf("%+v", m)
}
