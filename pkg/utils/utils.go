package utils

import (
	"net"
	"os"
)

func IsValidIPAddress(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	return true
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DeleteFile(str string) error {
	if err := os.Remove(str); err != nil {
		return err
	}
	return nil
}
