//go:build windows

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const createNoWindow = 0x08000000

const (
	seeMaskNoCloseProcess = 0x00000040
	swHide                = 0
)

var shellExecuteExW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle
	hProcess     windows.Handle
}

// HideCommandWindow prevents CLI helpers from opening a console from the GUI process.
func HideCommandWindow(command *exec.Cmd) {
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.HideWindow = true
	command.SysProcAttr.CreationFlags |= createNoWindow
}

// RunElevatedCommand launches a trusted executable through Windows UAC without a console.
func RunElevatedCommand(ctx context.Context, path, workingDir string, args ...string) (int, error) {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return 0, err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	parameters, err := windows.UTF16PtrFromString(joinWindowsArguments(args))
	if err != nil {
		return 0, err
	}
	var directory *uint16
	if workingDir != "" {
		directory, err = windows.UTF16PtrFromString(workingDir)
		if err != nil {
			return 0, err
		}
	}
	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: parameters,
		lpDirectory:  directory,
		nShow:        swHide,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))

	succeeded, _, callErr := shellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if succeeded == 0 {
		if errno, ok := callErr.(syscall.Errno); ok {
			return 0, fmt.Errorf("Windows elevation failed (code %d): %w", errno, callErr)
		}
		return 0, fmt.Errorf("Windows elevation failed: %w", callErr)
	}
	if info.hProcess == 0 {
		return 0, fmt.Errorf("Windows elevation did not return a process handle")
	}
	defer windows.CloseHandle(info.hProcess)

	for {
		waitResult, waitErr := windows.WaitForSingleObject(info.hProcess, 100)
		if waitErr != nil {
			return 0, fmt.Errorf("wait for elevated process: %w", waitErr)
		}
		if waitResult == windows.WAIT_OBJECT_0 {
			var exitCode uint32
			if err := windows.GetExitCodeProcess(info.hProcess, &exitCode); err != nil {
				return 0, fmt.Errorf("read elevated process exit code: %w", err)
			}
			return int(exitCode), nil
		}
		if waitResult != uint32(windows.WAIT_TIMEOUT) {
			return 0, fmt.Errorf("wait for elevated process returned status %d", waitResult)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}
}

func joinWindowsArguments(args []string) string {
	escaped := make([]string, len(args))
	for index, arg := range args {
		escaped[index] = syscall.EscapeArg(arg)
	}
	return strings.Join(escaped, " ")
}
