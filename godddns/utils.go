package godddns

import "net"

/*
	Private IP checking
*/

var privateIPNetworks = []net.IPNet{
	{
		IP:   net.ParseIP("10.0.0.0"),
		Mask: net.CIDRMask(8, 32),
	},
	{
		IP:   net.ParseIP("172.16.0.0"),
		Mask: net.CIDRMask(12, 32),
	},
	{
		IP:   net.ParseIP("192.168.0.0"),
		Mask: net.CIDRMask(16, 32),
	},
}

func isPrivateIpString(ipaddr string) bool {
	ip := net.ParseIP(ipaddr)
	if ip == nil {
		return false
	}

	return IsPrivateIP(ip)

}

func IsPrivateIP(ip net.IP) bool {
	for _, ipNet := range privateIPNetworks {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}
