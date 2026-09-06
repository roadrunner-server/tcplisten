//go:build linux || darwin || freebsd

package tcplisten

import "net"

const (
	IPV4 string = "tcp4"
	IPV6 string = "tcp6"
)

func createTCPListener(addr string) (net.Listener, error) {
	cfg := Config{
		ReusePort:   true,
		DeferAccept: false,
		FastOpen:    true,
	}

	/*
		Options we may have here:
		1. [::1]:8080 //ipv6
		2. [0:0:..]:8080 //ipv6
		3. 127.0.0.1:8080 //ipv4
		4. :8080 //ipv4
		5. [::]:8080 //ipv6
	*/
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// consider this is IPv4
	if host == "" {
		return cfg.NewListener(IPV4, addr)
	}

	return cfg.NewListener(netw(net.ParseIP(host)), addr)
}

// check if we are listening on the ipv6 or ipv4 address
func netw(addr net.IP) string {
	if addr.To4() == nil {
		return IPV6
	}
	return IPV4
}
