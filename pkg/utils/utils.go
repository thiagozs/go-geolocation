package utils

import (
	"net"
)

func IsValidIPAddress(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	return true
}
