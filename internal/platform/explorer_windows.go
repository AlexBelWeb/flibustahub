//go:build windows

package platform

import (
	"os/exec"
	"syscall"
)

func explorerQuotedPath(abs string) string {
	quoted := syscall.EscapeArg(abs)
	if quoted == "" {
		return `""`
	}
	if quoted[0] == '"' {
		return quoted
	}
	return `"` + quoted + `"`
}

func explorerSelectCmdLine(abs string) string {
	return "explorer.exe /select," + explorerQuotedPath(abs)
}

func explorerOpenCmdLine(abs string) string {
	return "explorer.exe " + explorerQuotedPath(abs)
}

func startExplorer(cmdLine string) error {
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdLine}
	return cmd.Start()
}

func revealPathOS(abs string) error {
	return startExplorer(explorerSelectCmdLine(abs))
}

func openDirOS(abs string) error {
	return startExplorer(explorerOpenCmdLine(abs))
}
