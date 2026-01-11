// Debug helper for testing install/uninstall flows.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/VoxDroid/krnr/internal/install"
)

func main() {
	if runtime.GOOS != "windows" {
		fmt.Println("This debug helper is intended for Windows runs only")
		return
	}
	tmp, err := os.MkdirTemp("", "debuginstall")
	if err != nil { panic(err) }
	defer func() { _ = os.RemoveAll(tmp) }()
	installDir := filepath.Join(tmp, "krnr")
	_ = os.MkdirAll(installDir, 0o755)
	src := filepath.Join(tmp, "srcbin")
	_ = os.WriteFile(src, []byte("binstuff"), 0o644)
	opts := install.Options{User:false, System:true, Path:installDir, From:src, DryRun:false, AddToPath:true}
	// test mode
	t := os.Setenv("KRNR_TEST_NO_SETX", "1")
	_ = t
	actions, err := install.ExecuteInstall(opts)
	fmt.Println("install actions:", actions, "err:", err)
	actions, err = install.Uninstall(false)
	fmt.Println("uninstall actions:", actions, "err:", err)
	info, err := os.Stat(installDir)
	if err != nil {
		fmt.Println("installDir missing as expected")
	} else {
		fmt.Println("installDir still exists: ", info.Name())
		ents, _ := os.ReadDir(installDir)
		for _, e := range ents { fmt.Println(" - entry:", e.Name()) }
	}
}
