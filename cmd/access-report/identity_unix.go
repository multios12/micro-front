//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func fileIdentity(info os.FileInfo) (string, string) {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprint(st.Dev), fmt.Sprint(st.Ino)
	}
	return "", ""
}
