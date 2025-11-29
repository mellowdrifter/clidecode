package clidecode

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// BirdClient represents a connection to a BIRD daemon via Unix socket
type BirdClient struct {
	SocketPath string
}

// query sends a command to the BIRD socket and returns the output
func (b *BirdClient) query(command string) (string, error) {
	return querySocket(b.SocketPath, command)
}

// RunCommand executes an arbitrary command on the BIRD socket
func (b *BirdClient) RunCommand(command string) (string, error) {
	return b.query(command)
}

// GetVersion returns the BIRD version string
func (b *BirdClient) GetVersion() (string, error) {
	out, err := b.query("show status")
	if err != nil {
		return "", err
	}

	// Output format:
	// BIRD 2.0.8
	// Router ID is ...
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		return lines[0], nil
	}
	return "", fmt.Errorf("empty status output")
}

// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
func (b *BirdClient) GetBGPTotal() (Totals, error) {
	var t Totals

	out, err := b.query("show route count")
	if err != nil {
		return t, err
	}

	// Parse output like:
	// 1024 of 1024 routes for 1024 networks in table master4
	// 2048 of 2048 routes for 2048 networks in table master6
	lines := strings.Split(out, "\n")

	// Regex to capture:
	// 1. Total routes (RIB) - the number before "routes" (or "of X routes")
	// 2. Networks (FIB) - the number before "networks"
	// Example: "2076414 of 2076414 routes for 1038207 networks"
	// We want 2076414 (RIB) and 1038207 (FIB)
	re := regexp.MustCompile(`(\d+)\s+of\s+\d+\s+routes\s+for\s+(\d+)\s+networks`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 3 {
			if strings.Contains(line, "master4") {
				t.V4Rib = stringToUint32(matches[1])
				t.V4Fib = stringToUint32(matches[2])
			} else if strings.Contains(line, "master6") {
				t.V6Rib = stringToUint32(matches[1])
				t.V6Fib = stringToUint32(matches[2])
			}
		}
	}

	return t, nil
}

// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
func (b *BirdClient) GetPeers() (Peers, error) {
	var p Peers

	out, err := b.query("show protocols")
	if err != nil {
		return p, err
	}

	lines := strings.Split(out, "\n")

	v4Configured := uint32(0)
	v4Established := uint32(0)
	v6Configured := uint32(0)
	v6Established := uint32(0)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		protocolName := fields[0]
		status := fields[5]

		// Skip non-BGP protocols
		if !strings.Contains(protocolName, "_v4") && !strings.Contains(protocolName, "_v6") {
			continue
		}

		// Skip system protocols
		if strings.Contains(line, "device1") || strings.Contains(line, "kernel1") {
			continue
		}

		if strings.Contains(protocolName, "_v4") {
			v4Configured++
			if status == "Established" {
				v4Established++
			}
		} else if strings.Contains(protocolName, "_v6") {
			v6Configured++
			if status == "Established" {
				v6Established++
			}
		}
	}

	p.V4c = v4Configured
	p.V4e = v4Established
	p.V6c = v6Configured
	p.V6e = v6Established

	return p, nil
}

// GetTotalSourceASNs returns total amount of unique ASNs
func (b *BirdClient) GetTotalSourceASNs() (ASNs, error) {
	var s ASNs

	// Get IPv4 source ASNs
	out4, err := b.query("show route primary table master4")
	if err != nil {
		return s, err
	}

	// Get IPv6 source ASNs
	out6, err := b.query("show route primary table master6")
	if err != nil {
		return s, err
	}

	// Extract source ASNs from output
	as4Set := extractSourceASNs(out4)
	as6Set := extractSourceASNs(out6)

	// Calculate total unique ASNs
	var as10 []string
	as10 = append(as10, as4Set...)
	as10 = append(as10, as6Set...)
	as10Set := stringSliceToSet(as10)

	// Calculate ASNs only in one address family
	as4Only := stringSliceDifference(as4Set, as6Set)
	as6Only := stringSliceDifference(as6Set, as4Set)
	asBoth := stringSliceIntersection(as4Set, as6Set)

	s.As4 = uint32(len(as4Set))
	s.As6 = uint32(len(as6Set))
	s.As10 = uint32(len(as10Set))
	s.As4Only = uint32(len(as4Only))
	s.As6Only = uint32(len(as6Only))
	s.AsBoth = uint32(len(asBoth))

	return s, nil
}

// extractSourceASNs extracts unique source ASNs from BIRD route output
func extractSourceASNs(output string) []string {
	re := regexp.MustCompile(`\[AS(\d+)[ie]?\]`)
	lines := strings.Split(output, "\n")

	asnMap := make(map[string]bool)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			asnMap[matches[1]] = true
		}
	}

	// Convert map to slice
	asns := make([]string, 0, len(asnMap))
	for asn := range asnMap {
		asns = append(asns, asn)
	}

	return asns
}

// GetROAs returns total amount of all ROA states
func (b *BirdClient) GetROAs() (Roas, error) {
	var r Roas

	// IPv4 ROA counts
	v4Valid, err := b.query("show route primary table master4 where roa_check(roa_v4) = ROA_VALID count")
	if err != nil {
		return r, err
	}
	r.V4v = extractRouteCount(v4Valid)

	v4Invalid, err := b.query("show route primary table master4 where roa_check(roa_v4) = ROA_INVALID count")
	if err != nil {
		return r, err
	}
	r.V4i = extractRouteCount(v4Invalid)

	v4Unknown, err := b.query("show route primary table master4 where roa_check(roa_v4) = ROA_UNKNOWN count")
	if err != nil {
		return r, err
	}
	r.V4u = extractRouteCount(v4Unknown)

	// IPv6 ROA counts
	v6Valid, err := b.query("show route primary table master6 where roa_check(roa_v6) = ROA_VALID count")
	if err != nil {
		return r, err
	}
	r.V6v = extractRouteCount(v6Valid)

	v6Invalid, err := b.query("show route primary table master6 where roa_check(roa_v6) = ROA_INVALID count")
	if err != nil {
		return r, err
	}
	r.V6i = extractRouteCount(v6Invalid)

	v6Unknown, err := b.query("show route primary table master6 where roa_check(roa_v6) = ROA_UNKNOWN count")
	if err != nil {
		return r, err
	}
	r.V6u = extractRouteCount(v6Unknown)

	return r, nil
}

// extractRouteCount extracts the route count from "show route count" output
func extractRouteCount(output string) uint32 {
	// Output format: "1024 of 1024 routes for 1024 network"
	fields := strings.Fields(output)
	if len(fields) > 0 {
		return stringToUint32(fields[0])
	}
	return 0
}

// GetInvalids returns a map of ASNs that are advertising RPKI invalid prefixes
func (b *BirdClient) GetInvalids() (map[string][]string, error) {
	inv := make(map[string][]string)
	num := regexp.MustCompile(`[\d]+`)

	// Get IPv4 invalids
	out4, err := b.query("show route primary table master4 where roa_check(roa_v4) = ROA_INVALID")
	if err != nil {
		return inv, err
	}

	lines := strings.Split(out4, "\n")
	for _, line := range lines {
		if prefix, asn := parseInvalidLine(line, num); asn != "" {
			inv[asn] = append(inv[asn], prefix)
		}
	}

	// Get IPv6 invalids
	out6, err := b.query("show route primary table master6 where roa_check(roa_v6) = ROA_INVALID")
	if err != nil {
		return inv, err
	}

	lines = strings.Split(out6, "\n")
	for _, line := range lines {
		if prefix, asn := parseInvalidLine(line, num); asn != "" {
			inv[asn] = append(inv[asn], prefix)
		}
	}

	return inv, nil
}

// parseInvalidLine extracts prefix and ASN from an invalid route line
func parseInvalidLine(line string, numRe *regexp.Regexp) (prefix, asn string) {
	// Example line: "192.0.2.0/24 via ... [AS64496i]"
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", ""
	}

	// First field is the prefix
	prefix = fields[0]

	// Find ASN in brackets
	idx := strings.Index(line, "[")
	if idx == -1 {
		return "", ""
	}
	asnMatch := numRe.FindString(line[idx:])
	if asnMatch != "" {
		asn = asnMatch
	}

	return prefix, asn
}

// GetMasks returns the total count of each mask value
func (b *BirdClient) GetMasks() ([]map[string]uint32, error) {
	v4 := make(map[string]uint32)
	v6 := make(map[string]uint32)

	// Get IPv4 routes
	out4, err := b.query("show route primary table master4")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out4, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "/") {
			parts := strings.Split(fields[0], "/")
			if len(parts) == 2 {
				v4[parts[1]]++
			}
		}
	}

	// Get IPv6 routes
	out6, err := b.query("show route primary table master6")
	if err != nil {
		return nil, err
	}

	lines = strings.Split(out6, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "/") {
			parts := strings.Split(fields[0], "/")
			if len(parts) == 2 {
				v6[parts[1]]++
			}
		}
	}

	return []map[string]uint32{v4, v6}, nil
}

// GetLargeCommunities returns the amount of prefixes that have large communities attached
func (b *BirdClient) GetLargeCommunities() (Large, error) {
	var l Large

	// IPv4 large communities
	out4, err := b.query("show route primary table master4 where bgp_large_community ~ [(*,*,*)]")
	if err != nil {
		return l, err
	}
	l.V4 = uint32(countRouteLines(out4))

	// IPv6 large communities
	out6, err := b.query("show route primary table master6 where bgp_large_community ~ [(*,*,*)]")
	if err != nil {
		return l, err
	}
	l.V6 = uint32(countRouteLines(out6))

	return l, nil
}

// countRouteLines counts the number of route lines in output
func countRouteLines(output string) int {
	if output == "" {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		// Count lines that start with an IP address
		if len(line) > 0 && (line[0] >= '0' && line[0] <= '9' || line[0] == ':') {
			count++
		}
	}
	return count
}

// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN
func (b *BirdClient) GetIPv4FromSource(asn uint32) ([]*net.IPNet, error) {
	cmd := fmt.Sprintf("show route primary table master4 where bgp_path ~ [= * %d =]", asn)
	out, err := b.query(cmd)
	if err != nil {
		return []*net.IPNet{}, err
	}

	return extractPrefixes(out), nil
}

// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN
func (b *BirdClient) GetIPv6FromSource(asn uint32) ([]*net.IPNet, error) {
	cmd := fmt.Sprintf("show route primary table master6 where bgp_path ~ [= * %d =]", asn)
	out, err := b.query(cmd)
	if err != nil {
		return nil, err
	}

	return extractPrefixes(out), nil
}

// extractPrefixes extracts IP prefixes from BIRD route output
func extractPrefixes(output string) []*net.IPNet {
	var ips []*net.IPNet
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "/") {
			_, ipNet, err := net.ParseCIDR(fields[0])
			if err == nil {
				ips = append(ips, ipNet)
			}
		}
	}

	return ips
}

// GetASPathFromIP will return the AS path, as well as as-set if any from a source IP
func (b *BirdClient) GetASPathFromIP(ip net.IP) (ASPath, bool, error) {
	var aspath ASPath

	cmd := fmt.Sprintf("show route primary all for %s", ip.String())
	out, err := b.query(cmd)
	if err != nil {
		return aspath, false, err
	}

	// If no route exists, no as-path will exist
	if out == "" || !strings.Contains(out, "BGP.as_path:") {
		return aspath, false, nil
	}

	// Extract AS path from output
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "BGP.as_path:") {
			// Extract the path after "BGP.as_path:"
			parts := strings.SplitN(line, "BGP.as_path:", 2)
			if len(parts) == 2 {
				path, set := decodeASPaths(strings.TrimSpace(parts[1]))
				aspath.Path = path
				aspath.Set = set
				return aspath, true, nil
			}
		}
	}

	return aspath, false, nil
}

// GetRoute will return the current FIB entry, if any, from a source IP
func (b *BirdClient) GetRoute(ip net.IP) (*net.IPNet, bool, error) {
	cmd := fmt.Sprintf("show route primary for %s", ip.String())
	out, err := b.query(cmd)
	if err != nil {
		return nil, false, err
	}

	if out == "" {
		return nil, false, nil
	}

	// Find the line containing the prefix
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "/") {
			_, ipNet, err := net.ParseCIDR(fields[0])
			if err == nil {
				return ipNet, true, nil
			}
		}
	}

	return nil, false, nil
}

// GetOriginFromIP will return the origin ASN from a source IP
func (b *BirdClient) GetOriginFromIP(ip net.IP) (uint32, bool, error) {
	cmd := fmt.Sprintf("show route primary all for %s", ip.String())
	out, err := b.query(cmd)
	if err != nil {
		return 0, false, err
	}

	if out == "" {
		return 0, false, nil
	}

	// Look for BGP.as_path line and extract the last ASN
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "BGP.as_path:") {
			parts := strings.SplitN(line, "BGP.as_path:", 2)
			if len(parts) == 2 {
				// Remove AS-SETs (in braces) and get the last ASN
				pathStr := strings.TrimSpace(parts[1])
				pathStr = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(pathStr, "")

				fields := strings.Fields(pathStr)
				if len(fields) > 0 {
					lastASN := fields[len(fields)-1]
					num := regexp.MustCompile(`[0-9]+`)
					o := num.FindString(lastASN)
					if o != "" {
						source, err := strconv.Atoi(o)
						if err != nil {
							return 0, true, err
						}
						return uint32(source), true, nil
					}
				}
			}
		}
	}

	return 0, false, nil
}

// GetROA will return the ROA status from a prefix and ASN
func (b *BirdClient) GetROA(prefix *net.IPNet, asn uint32) (int, bool, error) {
	var table string
	if strings.Contains(prefix.String(), ":") {
		table = "roa_v6"
	} else {
		table = "roa_v4"
	}

	cmd := fmt.Sprintf("eval roa_check(%s, %s, %d)", table, prefix, asn)
	out, err := b.query(cmd)
	if err != nil {
		return 0, false, err
	}

	// Parse the enum value from output like "(enum 35)1"
	if len(out) > 0 {
		val := out[len(out)-1:]

		statuses := map[string]int{
			"0": RUnknown,
			"1": RValid,
			"2": RInvalid,
		}

		if status, ok := statuses[val]; ok {
			return status, true, nil
		}
	}

	return 0, false, nil
}

// GetVRPs will return all Validated ROA Payloads for an ASN
func (b *BirdClient) GetVRPs(asn uint32) ([]VRP, error) {
	var VRPs []VRP

	// Get IPv4 VRPs
	cmd4 := fmt.Sprintf("show route all table roa_v4 where net.asn=%d", asn)
	out4, err := b.query(cmd4)
	if err != nil {
		return VRPs, err
	}

	if out4 != "" {
		vrps, err := parseVRPs(out4)
		if err != nil {
			return nil, err
		}
		VRPs = append(VRPs, vrps...)
	}

	// Get IPv6 VRPs
	cmd6 := fmt.Sprintf("show route all table roa_v6 where net.asn=%d", asn)
	out6, err := b.query(cmd6)
	if err != nil {
		return VRPs, err
	}

	if out6 != "" {
		vrps, err := parseVRPs(out6)
		if err != nil {
			return nil, err
		}
		VRPs = append(VRPs, vrps...)
	}

	return VRPs, nil
}

// parseVRPs parses VRP entries from BIRD output
func parseVRPs(output string) ([]VRP, error) {
	var vrps []VRP
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Look for lines with prefix and max length like "192.0.2.0/24-24"
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "-") {
			parts := strings.Split(fields[0], "-")
			if len(parts) == 2 {
				_, prefix, err := net.ParseCIDR(parts[0])
				if err != nil {
					return nil, err
				}

				max, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, err
				}

				vrps = append(vrps, VRP{Prefix: prefix, Max: max})
			}
		}
	}

	return vrps, nil
}

// Helper functions for string slice operations

// stringSliceToSet returns a deduplicated slice of strings
func stringSliceToSet(slice []string) []string {
	set := make(map[string]bool)
	for _, s := range slice {
		set[s] = true
	}

	result := make([]string, 0, len(set))
	for s := range set {
		result = append(result, s)
	}
	return result
}

// stringSliceDifference returns elements in first but not in second
func stringSliceDifference(first, second []string) []string {
	secondSet := make(map[string]bool)
	for _, s := range second {
		secondSet[s] = true
	}

	var result []string
	for _, s := range first {
		if !secondSet[s] {
			result = append(result, s)
		}
	}
	return result
}

// stringSliceIntersection returns elements in both slices
func stringSliceIntersection(first, second []string) []string {
	secondSet := make(map[string]bool)
	for _, s := range second {
		secondSet[s] = true
	}

	var result []string
	for _, s := range first {
		if secondSet[s] {
			result = append(result, s)
		}
	}
	return result
}

// stringToUint32 converts a string to uint32
func stringToUint32(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(val)
}
