package util

import (
	"fmt"
	"net"
)

func GetBackupLogsDir() string {
	return "/var/log/eci-container-logs"
}

func GetAdditionalIPs(interfaceName string) ([]string, error) {
	var ips []string

	byName, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	addresses, err := byName.Addrs()
	if err != nil {
		return nil, err
	}

	for _, v := range addresses {
		ip, _, err := net.ParseCIDR(v.String())
		if err != nil {
			continue
		}

		if IsIPv4(ip) {
			ips = append(ips, ip.String())

			break
		}
	}

	for _, v := range addresses {
		ip, _, err := net.ParseCIDR(v.String())
		if err != nil {
			continue
		}

		if IsGlobalUnicastIPv6(ip) {
			ips = append(ips, ip.String())

			break
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return ips, nil
}

// IsIPv4 returns if netIP is IPv4.
func IsIPv4(netIP net.IP) bool {
	return netIP != nil && netIP.To4() != nil
}

// IsIPv6 returns if netIP is IPv6.
func IsIPv6(netIP net.IP) bool {
	return netIP != nil && netIP.To4() == nil
}

// global unicast address
func IsGlobalUnicastIPv6(ip net.IP) bool {
	if IsIPv6(ip) && ip.IsGlobalUnicast() {
		return true
	}

	return false
}
