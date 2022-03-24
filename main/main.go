package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	// forkednet "k8s.io/utils/internal/third_party/forked/golang/net"
)

// var ParseIPSloppy = forkednet.ParseIP
var CIDR string

func main() {
	if len(os.Args) < 3 {
		klog.Error("Pass time in milli secs to sleep and CIDR")
		return
	}
	sleepTime := os.Args[1] + "ms"
	CIDR = os.Args[2]
	duration, err := time.ParseDuration(sleepTime)
	if err != nil {
		klog.Error("Failed to parse time:", err)
		return
	}
	for {
		ips, err := fetchNodeIPs()
		if err != nil {
			klog.Error("fetching Node Ips failed:", err)
		} else {
			klog.Error("Node IPs:", ips)
		}
		time.Sleep(duration)
	}
}

func fetchNodeIPs() (ips []net.IP, err error) {
	nodeAddress, err := getAllLocalAddresses()
	if err != nil {
		return nil, fmt.Errorf("error listing LOCAL type addresses from host, error: %v", err)
	}

	bindedAddress, err := bindedIPs()
	if err != nil {
		return nil, err
	}
	ipset := nodeAddress.Difference(bindedAddress)
	invalidIP := false
	for _, ipStr := range ipset.UnsortedList() {
		a := ParseIP(ipStr)
		ips = append(ips, a)
		_, ipnet, err := net.ParseCIDR(CIDR)
		if err != nil {
			klog.Error("Could not parse cidr, error", err)
		}
		if ipnet != nil && ipnet.Contains(a) && !invalidIP {
			invalidIP = true
			klog.Error("Node IP contains cluster ip:", a)
		}

	}
	if invalidIP {
		klog.Errorf("NodeAddresses: %+v,\n BindAddresses:%+v,\n Diff: %+v", nodeAddress, bindedAddress, ips)
	}

	return ips, nil
}

func getAllLocalAddresses() (sets.String, error) {
	addr, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("Could not get addresses: %v", err)
	}
	return AddressSet(isValidForSet, addr), nil
}

func bindedIPs() (sets.String, error) {
	return GetLocalAddresses("kube-ipvs0")
}

func GetLocalAddresses(dev string) (sets.String, error) {
	ifi, err := net.InterfaceByName(dev)
	if err != nil {
		return nil, fmt.Errorf("Could not get interface %s: %v", dev, err)
	}
	addr, err := ifi.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Can't get addresses from %s: %v", ifi.Name, err)
	}
	return AddressSet(isValidForSet, addr), nil
}

func AddressSet(isValid func(ip net.IP) bool, addrs []net.Addr) sets.String {
	ips := sets.NewString()
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPAddr:
			ip = v.IP
		case *net.IPNet:
			ip = v.IP
		default:
			continue
		}
		if isValid(ip) {
			ips.Insert(ip.String())
		}
	}
	return ips
}

func isValidForSet(ip net.IP) bool {
	if IsIPv6(ip) {
		return false
	}
	// if h.isIPv6 && ip.IsLinkLocalUnicast() {
	// 	return false
	// }
	if ip.IsLoopback() {
		return false
	}
	return true
}

func IsIPv6(netIP net.IP) bool {
	return netIP != nil && netIP.To4() == nil
}

