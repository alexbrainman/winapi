// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winapi_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func testTlsThread(t *testing.T, tlsidx uint32) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadId := winapi.GetCurrentThreadId()

	want := uintptr(threadId)
	err := winapi.TlsSetValue(tlsidx, want)
	if err != nil {
		t.Fatal(err)
	}
	have, err := winapi.TlsGetValue(tlsidx)
	if err != nil {
		t.Fatal(err)
	}
	if want != have {
		t.Errorf("threadid=%d: unexpected tls data %d, want %d", threadId, have, want)
	}
}

func TestTls(t *testing.T) {
	tlsidx, err := winapi.TlsAlloc()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := winapi.TlsFree(tlsidx)
		if err != nil {
			t.Fatal(err)
		}
	}()

	const threadCount = 20

	done := make(chan bool)
	for i := 0; i < threadCount; i++ {
		go func() {
			defer func() {
				done <- true
			}()
			testTlsThread(t, tlsidx)
		}()
	}
	for i := 0; i < threadCount; i++ {
		<-done
	}
}

func runIcacls(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("icacls", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("icacls failed: %v\n%v", err, string(out))
	}
	return string(out)
}

func adjustACL(t *testing.T, path string) {
	// as described in
	// https://stackoverflow.com/questions/17536692/resetting-file-security-to-inherit-after-a-movefile-operation
	var acl winapi.ACL
	err := winapi.InitializeAcl(&acl, uint32(unsafe.Sizeof(acl)), winapi.ACL_REVISION)
	if err != nil {
		t.Fatal(err)
	}
	errno := winapi.SetNamedSecurityInfo(syscall.StringToUTF16Ptr(path), winapi.SE_FILE_OBJECT, winapi.DACL_SECURITY_INFORMATION|winapi.UNPROTECTED_DACL_SECURITY_INFORMATION, nil, nil, &acl, nil)
	if errno != 0 {
		t.Fatalf("SetNamedSecurityInfo failed: %v", syscall.Errno(errno))
	}
}

func runGetACL(t *testing.T, path string) string {
	t.Helper()
	cmd := fmt.Sprintf(`Get-Acl "%s" | Select -expand AccessToString`, path)
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	if err != nil {
		t.Fatalf("Get-Acl failed: %v\n%v", err, string(out))
	}
	return string(out)
}

func TestIssue22343(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "TestIssue22343")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	newtmpdir := filepath.Join(tmpdir, "tmp")
	err = os.Mkdir(newtmpdir, 0777)
	if err != nil {
		t.Fatal(err)
	}

	// When TestIssue22343/tmp directory is created, it will have
	// the same security attributes as TestIssue22343.
	// Add Guest account full access to TestIssue22343/tmp - this
	// will make all files created in TestIssue22343/tmp have different
	// security attributes to the files created in TestIssue22343.
	runIcacls(t, newtmpdir,
		"/inheritance:r",  // breaks the inheritance from the directory above
		"/grant:r",        // clears the ACL
		"guest:(oi)(ci)f", // Guest user will have full access
	)

	src := filepath.Join(tmpdir, "main.go")
	err = ioutil.WriteFile(src, []byte("package main; func main() { }\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmpdir, "main.exe")
	cmd := exec.Command("go", "build", "-o", exe, src)
	cmd.Env = append(os.Environ(),
		"TMP="+newtmpdir,
		"TEMP="+newtmpdir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go command failed: %v\n%v", err, string(out))
	}

	adjustACL(t, exe)

	// exe file is expected to have the same security atributes as the src.
	if got, expected := runGetACL(t, exe), runGetACL(t, src); got != expected {
		t.Fatalf("expected Get-Acl output of \n%v\n, got \n%v\n", expected, got)
	}
}
