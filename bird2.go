package clidecode

import (
	"net"
	"strings"
)

// Bird2Conn represents a connection to a BIRD 2 daemon
type Bird2Conn struct {
	BirdClient
}

// NewBird2Conn creates a new Bird2Conn with the default socket path
func NewBird2Conn() *Bird2Conn {
	return &Bird2Conn{
		BirdClient: BirdClient{
			SocketPath: "/run/bird.ctl",
		},
	}
}

// NewBird2ConnWithSocket creates a new Bird2Conn with a custom socket path
func NewBird2ConnWithSocket(socketPath string) *Bird2Conn {
	return &Bird2Conn{
		BirdClient: BirdClient{
			SocketPath: socketPath,
		},
	}
}

// decodeASPaths will return a slice of AS & AS-Sets from a string as-path output.
func decodeASPaths(in string) ([]uint32, []uint32) {
	if strings.ContainsAny(in, "{}") {
		in = strings.Replace(in, "{", "{ ", 1)
		in = strings.Replace(in, "}", " }", 1)
	}
	paths := strings.Fields(in)
	var path, set []uint32

	// Need to separate as-set
	var isSet bool
	for _, as := range paths {
		if strings.ContainsAny(as, "{}") {
			isSet = true
			continue
		}

		switch {
		case isSet == false:
			path = append(path, stringToUint32(as))
		case isSet == true:
			set = append(set, stringToUint32(as))
		}
	}

	return path, set
}

// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
func (b Bird2Conn) GetBGPTotal() (Totals, error) {
	return b.BirdClient.GetBGPTotal()
}

// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
func (b Bird2Conn) GetPeers() (Peers, error) {
	return b.BirdClient.GetPeers()
}

// GetTotalSourceASNs returns total amount of unique ASNs
func (b Bird2Conn) GetTotalSourceASNs() (ASNs, error) {
	return b.BirdClient.GetTotalSourceASNs()
}

// GetROAs returns total amount of all ROA states
func (b Bird2Conn) GetROAs() (Roas, error) {
	return b.BirdClient.GetROAs()
}

// GetInvalids returns a map of ASNs that are advertising RPKI invalid prefixes
func (b Bird2Conn) GetInvalids() (map[string][]string, error) {
	return b.BirdClient.GetInvalids()
}

// GetMasks returns the total count of each mask value
func (b Bird2Conn) GetMasks() ([]map[string]uint32, error) {
	return b.BirdClient.GetMasks()
}

// GetLargeCommunities returns the amount of prefixes that have large communities attached
func (b Bird2Conn) GetLargeCommunities() (Large, error) {
	return b.BirdClient.GetLargeCommunities()
}

// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN
func (b Bird2Conn) GetIPv4FromSource(asn uint32) ([]*net.IPNet, error) {
	return b.BirdClient.GetIPv4FromSource(asn)
}

// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN
func (b Bird2Conn) GetIPv6FromSource(asn uint32) ([]*net.IPNet, error) {
	return b.BirdClient.GetIPv6FromSource(asn)
}

// GetASPathFromIP will return the AS path, as well as as-set if any from a source IP
func (b Bird2Conn) GetASPathFromIP(ip net.IP) (ASPath, bool, error) {
	return b.BirdClient.GetASPathFromIP(ip)
}

// GetRoute will return the current FIB entry, if any, from a source IP
func (b Bird2Conn) GetRoute(ip net.IP) (*net.IPNet, bool, error) {
	return b.BirdClient.GetRoute(ip)
}

// GetOriginFromIP will return the origin ASN from a source IP
func (b Bird2Conn) GetOriginFromIP(ip net.IP) (uint32, bool, error) {
	return b.BirdClient.GetOriginFromIP(ip)
}

// GetROA will return the ROA status from a prefix and ASN
func (b Bird2Conn) GetROA(prefix *net.IPNet, asn uint32) (int, bool, error) {
	return b.BirdClient.GetROA(prefix, asn)
}

// GetVRPs will return all Validated ROA Payloads for an ASN
func (b Bird2Conn) GetVRPs(asn uint32) ([]VRP, error) {
	return b.BirdClient.GetVRPs(asn)
}
