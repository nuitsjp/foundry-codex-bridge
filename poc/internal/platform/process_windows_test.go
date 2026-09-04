//go:build windows

package platform

import (
	"os/exec"
	"testing"
)

func TestHideCommandWindowConfiguresWindowsProcessFlags(t *testing.T) {
	command := exec.Command("cmd.exe", "/c", "exit", "0")
	HideCommandWindow(command)

	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatal("HideCommandWindow() did not hide the child window")
	}
	if command.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("HideCommandWindow() did not set CREATE_NO_WINDOW")
	}
}

func TestJoinWindowsArgumentsQuotesSpacesAndQuotes(t *testing.T) {
	got := joinWindowsArguments([]string{`C:\Program Files\node\ocx.mjs`, "service", `say "yes"`})
	want := `"C:\Program Files\node\ocx.mjs" service "say \"yes\""`
	if got != want {
		t.Fatalf("joinWindowsArguments() = %q, want %q", got, want)
	}
}
