package validation

import (
	"net"
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation"
)

func IsValidAddress(addr string) []string {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return []string{err.Error()}
	}

	var errs []string
	if host != "" && host != "localhost" {
		if ip := net.ParseIP(host); ip == nil {
			errs = append(errs, "invalid textual representation of an IP address")
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		errs = append(errs, validation.IsValidPortNum(port)...)
	}

	return errs
}
