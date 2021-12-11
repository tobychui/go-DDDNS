package godddns

import (
	"net"
	"strings"
)

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
	{
		IP:   net.ParseIP("127.0.0.0"),
		Mask: net.CIDRMask(8, 32),
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

/*
	Trim the port number from returned net.IP
*/

func trimIpPort(ipWithPort string) string {
	if strings.Contains(ipWithPort, ":") {
		//from LAN or testing environment which contains the port after the reflected IP addr, trim that part
		tmp := strings.Split(ipWithPort, ":")
		result := tmp[0]
		return result
	} else {
		return ipWithPort
	}
}
