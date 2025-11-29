package clidecode

import (
	"net"
)

// Bird3Conn represents a connection to a BIRD 3 daemon
type Bird3Conn struct {
	BirdClient
}

// NewBird3Conn creates a new Bird3Conn with the default socket path
func NewBird3Conn() *Bird3Conn {
	return &Bird3Conn{
		BirdClient: BirdClient{
			SocketPath: "/run/bird3.ctl",
		},
	}
}

// NewBird3ConnWithSocket creates a new Bird3Conn with a custom socket path
func NewBird3ConnWithSocket(socketPath string) *Bird3Conn {
	return &Bird3Conn{
		BirdClient: BirdClient{
			SocketPath: socketPath,
		},
	}
}

// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
func (b Bird3Conn) GetBGPTotal() (Totals, error) {
	return b.BirdClient.GetBGPTotal()
}

// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
func (b Bird3Conn) GetPeers() (Peers, error) {
	return b.BirdClient.GetPeers()
}

// GetTotalSourceASNs returns total amount of unique ASNs
func (b Bird3Conn) GetTotalSourceASNs() (ASNs, error) {
	return b.BirdClient.GetTotalSourceASNs()
}

// GetROAs returns total amount of all ROA states
func (b Bird3Conn) GetROAs() (Roas, error) {
	return b.BirdClient.GetROAs()
}

// GetInvalids returns a map of ASNs that are advertising RPKI invalid prefixes
func (b Bird3Conn) GetInvalids() (map[string][]string, error) {
	return b.BirdClient.GetInvalids()
}

// GetMasks returns the total count of each mask value
func (b Bird3Conn) GetMasks() ([]map[string]uint32, error) {
	return b.BirdClient.GetMasks()
}

// GetLargeCommunities returns the amount of prefixes that have large communities attached
func (b Bird3Conn) GetLargeCommunities() (Large, error) {
	return b.BirdClient.GetLargeCommunities()
}

// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN
func (b Bird3Conn) GetIPv4FromSource(asn uint32) ([]*net.IPNet, error) {
	return b.BirdClient.GetIPv4FromSource(asn)
}

// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN
func (b Bird3Conn) GetIPv6FromSource(asn uint32) ([]*net.IPNet, error) {
	return b.BirdClient.GetIPv6FromSource(asn)
}

// GetASPathFromIP will return the AS path, as well as as-set if any from a source IP
func (b Bird3Conn) GetASPathFromIP(ip net.IP) (ASPath, bool, error) {
	return b.BirdClient.GetASPathFromIP(ip)
}

// GetRoute will return the current FIB entry, if any, from a source IP
func (b Bird3Conn) GetRoute(ip net.IP) (*net.IPNet, bool, error) {
	return b.BirdClient.GetRoute(ip)
}

// GetOriginFromIP will return the origin ASN from a source IP
func (b Bird3Conn) GetOriginFromIP(ip net.IP) (uint32, bool, error) {
	return b.BirdClient.GetOriginFromIP(ip)
}

// GetROA will return the ROA status from a prefix and ASN
func (b Bird3Conn) GetROA(prefix *net.IPNet, asn uint32) (int, bool, error) {
	return b.BirdClient.GetROA(prefix, asn)
}

// GetVRPs will return all Validated ROA Payloads for an ASN
func (b Bird3Conn) GetVRPs(asn uint32) ([]VRP, error) {
	return b.BirdClient.GetVRPs(asn)
}
