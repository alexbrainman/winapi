package main_test

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/alexbrainman/winapi"
)

var sleepBetweenRuns = flag.Duration("sleep", 0, "sleep between exe runs (defaults to no sleep)")

func buildGoExe(t *testing.T, tmpdir string) string {
	src := filepath.Join(tmpdir, "a.go")
	err := ioutil.WriteFile(src, []byte(`package main; func main() {}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmpdir, "a_go.exe")
	cmd := exec.Command("go", "build", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building test executable failed: %s %s", err, out)
	}
	return exe
}

func buildCExe(t *testing.T, tmpdir string) string {
	src := filepath.Join(tmpdir, "a.c")
	err := ioutil.WriteFile(src, []byte(csrc), 0644)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmpdir, "a_c.exe")
	cmd := exec.Command("gcc", "-o", exe, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building test executable failed: %s %s", err, out)
	}
	return exe
}

func dupStdHandle(stdh int) (syscall.Handle, error) {
	h, _ := syscall.GetStdHandle(stdh)
	p, _ := syscall.GetCurrentProcess()
	var dup syscall.Handle
	err := syscall.DuplicateHandle(p, h, p, &dup, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	if err != nil {
		return 0, os.NewSyscallError("DuplicateHandle", err)
	}
	return dup, nil
}

func runExe(argv0 string) error {
	argv0p, err := syscall.UTF16PtrFromString(argv0)
	if err != nil {
		return err
	}

	si := new(syscall.StartupInfo)
	si.Cb = uint32(unsafe.Sizeof(*si))
	si.Flags = syscall.STARTF_USESTDHANDLES
	si.StdInput, err = dupStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return err
	}
	si.StdOutput, err = dupStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return err
	}
	si.StdErr, err = dupStdHandle(syscall.STD_ERROR_HANDLE)
	if err != nil {
		return err
	}

	pi := new(syscall.ProcessInformation)

	flags := uint32(syscall.CREATE_UNICODE_ENVIRONMENT)
	env := []uint16{0, 0}
	err = syscall.CreateProcess(argv0p, nil, nil, nil, true, flags, &env[0], nil, si, pi)
	if err != nil {
		return os.NewSyscallError("CreateProcess", err)
	}

	syscall.CloseHandle(pi.Thread)
	syscall.CloseHandle(si.StdErr)
	syscall.CloseHandle(si.StdOutput)
	syscall.CloseHandle(si.StdInput)

	h := pi.Process

	s, err := syscall.WaitForSingleObject(h, syscall.INFINITE)
	switch s {
	case syscall.WAIT_OBJECT_0:
		break
	case syscall.WAIT_FAILED:
		return os.NewSyscallError("WaitForSingleObject", err)
	default:
		return errors.New("Unexpected result from WaitForSingleObject")
	}

	var ec uint32
	err = syscall.GetExitCodeProcess(h, &ec)
	if err != nil {
		return os.NewSyscallError("GetExitCodeProcess", err)
	}

	if *sleepBetweenRuns != 0 {
		time.Sleep(*sleepBetweenRuns)
	}

	err = syscall.CloseHandle(h)
	if err != nil {
		return os.NewSyscallError("CloseHandle", err)
	}
	return nil
}

func deleteFile(name string) error {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	err = syscall.DeleteFile(p)
	if err != nil {
		return os.NewSyscallError("DeleteFile", err)
	}
	return nil
}

func testGoExe(t *testing.T, dstexe, srcexe string) {
	for i := 0; i < 100; i++ {
		err := winapi.CopyFile(syscall.StringToUTF16Ptr(srcexe), syscall.StringToUTF16Ptr(dstexe), true)
		if err != nil {
			t.Errorf("iteration %d failed: %v", i, os.NewSyscallError("CopyFile", err))
			return
		}

		err = runExe(dstexe)
		if err != nil {
			t.Errorf("iteration %d failed: %v", i, err)
			return
		}

		err = deleteFile(dstexe)
		if err != nil {
			t.Errorf("iteration %d failed: %v", i, err)
			return
		}
	}
}

func testCExe(t *testing.T, cexe, dstexe, srcexe string) {
	ms := strconv.FormatInt(sleepBetweenRuns.Milliseconds(), 10)
	cmd := exec.Command(cexe, dstexe, srcexe, ms)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running c exe failed: %s %s", err, out)
	}
}

// Test for https://golang.org/issue/25965
func TestIssue25965(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "TestIssue25965")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	exe := buildGoExe(t, tmpdir)
	dstexe := filepath.Join(tmpdir, "a2.exe")
	testGoExe(t, dstexe, exe)

	cexe := buildCExe(t, tmpdir)
	testCExe(t, cexe, dstexe, exe)
}

var csrc = `
#include <windows.h>
#include <tchar.h>
#include <stdio.h>

HANDLE dupStdHandle(int stdh) {
	HANDLE h, p, dup;
	h = GetStdHandle(stdh);
	p = GetCurrentProcess();
	if (!DuplicateHandle(p, h, p, &dup, 0, TRUE, DUPLICATE_SAME_ACCESS))
	{
		printf("DuplicateHandle failed (%d)\n", GetLastError());
		return INVALID_HANDLE_VALUE;
	}
	return dup;
}

BOOL runExe(char* argv0, int ms) {
	STARTUPINFO si;
	PROCESS_INFORMATION pi;
	UINT16 env[2];
	HANDLE h;
	DWORD ec;

	ZeroMemory(&si, sizeof(si));
	si.cb = sizeof(si);
	si.dwFlags = STARTF_USESTDHANDLES;
	if (!(si.hStdInput = dupStdHandle(STD_INPUT_HANDLE)))
	{
		return FALSE;
	}
	if (!(si.hStdOutput = dupStdHandle(STD_OUTPUT_HANDLE)))
	{
		return FALSE;
	}
	if (!(si.hStdError = dupStdHandle(STD_ERROR_HANDLE)))
	{
		return FALSE;
	}

	ZeroMemory(&pi, sizeof(pi));

	ZeroMemory(&env, sizeof(env));

	if (!CreateProcess(argv0, NULL, NULL, NULL, TRUE, CREATE_UNICODE_ENVIRONMENT, &env[0], NULL, &si, &pi)) {
		printf("CreateProcess failed (%d)\n", GetLastError());
		return FALSE;
	}

	CloseHandle(pi.hThread);
	CloseHandle(si.hStdError);
	CloseHandle(si.hStdOutput);
	CloseHandle(si.hStdInput);

	h = pi.hProcess;

	if (WaitForSingleObject(h, INFINITE) != WAIT_OBJECT_0) {
		printf("WaitForSingleObject failed (%d)\n", GetLastError());
		return FALSE;
	}

	if (!GetExitCodeProcess(h, &ec)) {
		printf("GetExitCodeProcess failed (%d)\n", GetLastError());
		return FALSE;
	}

	if (ms > 0) {
		Sleep(ms);
	}

	if (!CloseHandle(h)) {
		printf("CloseHandle failed (%d)\n", GetLastError());
		return FALSE;
	}

	return TRUE;
}

BOOL testGoExe(char* dstexe, char* srcexe, int ms) {
	int i;

	for (i = 0; i < 100; i++) {
		if (!CopyFile(srcexe, dstexe, FALSE))
		{
			printf("iteration %d: CopyFile failed (%d)\n", i, GetLastError());
			return FALSE;
		}

		if (!runExe(dstexe, ms))
		{
			printf("during iteration %d\n", i);
			return FALSE;
		}

		if (!DeleteFile(dstexe))
		{
			printf("iteration %d: DeleteFile failed (%d)\n", i, GetLastError());
			return FALSE;
		}
	}
	return TRUE;
}

int main(int argc, char** argv)
{
	char *dstexe = argv[1];
	char *srcexe = argv[2];
	int ms = atoi(argv[3]);
	if (testGoExe(dstexe, srcexe, ms))
	{
		return 0;
	}
	else
	{
		return 1;
	}
}
`
