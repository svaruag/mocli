package config

import (
	"fmt"
	"os"
	"runtime"
)

func validatePrivateFile(path string, mode os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if mode.Perm()&0o077 != 0 {
		return fmt.Errorf("insecure file permissions on %s (mode %o); run chmod 600", path, mode.Perm())
	}
	return nil
}
