package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mellowdrifter/clidecode"
)

func main() {
	fmt.Println("BIRD Socket Connection Test & Interactive Tool")
	fmt.Println("===============================================")

	// Potential socket paths to check
	socketPaths := []string{
		"/run/bird.ctl",
		"/run/bird/bird.ctl",
		"/run/bird3.ctl",
		"/var/run/bird.ctl",
		"/var/run/bird3.ctl",
	}

	var client *clidecode.BirdClient
	var socketPath string

	// Find socket
	for _, path := range socketPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		socketPath = path
		break
	}

	if socketPath == "" {
		fmt.Println("❌ No BIRD sockets found in standard locations.")
		return
	}

	fmt.Printf("✅ Found socket at: %s\n", socketPath)
	client = &clidecode.BirdClient{SocketPath: socketPath}

	// Get Version
	if ver, err := client.GetVersion(); err == nil {
		fmt.Printf("   Daemon: %s\n", ver)
	} else {
		fmt.Printf("   Error getting version: %v\n", err)
	}
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		printMenu()
		fmt.Print("\nSelect an option: ")
		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		if choice == "0" || choice == "q" {
			fmt.Println("Exiting...")
			break
		}

		err := handleChoice(choice, client, scanner)
		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
		}

		fmt.Println("\nPress Enter to continue...")
		scanner.Scan()
	}
}

func printMenu() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println(" 1. GetBGPTotal (RIB/FIB counts)")
	fmt.Println(" 2. GetPeers (Peer counts)")
	fmt.Println(" 3. GetTotalSourceASNs (Unique ASN counts)")
	fmt.Println(" 4. GetMasks (Prefix length distribution)")
	fmt.Println(" 5. GetROAs (ROA state counts)")
	fmt.Println(" 6. GetLargeCommunities (Large community counts)")
	fmt.Println(" 7. GetIPv4FromSource (Prefixes by Origin ASN)")
	fmt.Println(" 8. GetIPv6FromSource (Prefixes by Origin ASN)")
	fmt.Println(" 9. GetOriginFromIP (Origin ASN for IP)")
	fmt.Println("10. GetASPathFromIP (AS Path for IP)")
	fmt.Println("11. GetRoute (FIB entry for IP)")
	fmt.Println("12. GetROA (ROA status for IP/ASN)")
	fmt.Println("13. GetVRPs (VRPs for ASN)")
	fmt.Println("14. GetInvalids (RPKI Invalid prefixes)")
	fmt.Println(" 0. Exit")
}

func handleChoice(choice string, client *clidecode.BirdClient, scanner *bufio.Scanner) error {
	fmt.Println(strings.Repeat("-", 50))

	switch choice {
	case "1":
		totals, err := client.GetBGPTotal()
		if err != nil {
			return err
		}
		fmt.Printf("IPv4 RIB: %d, FIB: %d\n", totals.V4Rib, totals.V4Fib)
		fmt.Printf("IPv6 RIB: %d, FIB: %d\n", totals.V6Rib, totals.V6Fib)

	case "2":
		peers, err := client.GetPeers()
		if err != nil {
			return err
		}
		fmt.Printf("IPv4 Peers: %d configured, %d established\n", peers.V4c, peers.V4e)
		fmt.Printf("IPv6 Peers: %d configured, %d established\n", peers.V6c, peers.V6e)

	case "3":
		asns, err := client.GetTotalSourceASNs()
		if err != nil {
			return err
		}
		fmt.Printf("AS4: %d, AS6: %d, AS10: %d\n", asns.As4, asns.As6, asns.As10)
		fmt.Printf("AS4Only: %d, AS6Only: %d, ASBoth: %d\n", asns.As4Only, asns.As6Only, asns.AsBoth)

	case "4":
		masks, err := client.GetMasks()
		if err != nil {
			return err
		}

		// Helper to sort and print masks
		printSortedMasks := func(title string, m map[string]uint32) {
			fmt.Println(title)
			type kv struct {
				Key   string
				Value uint32
			}
			var ss []kv
			for k, v := range m {
				ss = append(ss, kv{k, v})
			}
			// Sort by Value descending
			sort.Slice(ss, func(i, j int) bool {
				return ss[i].Value > ss[j].Value
			})
			for _, kv := range ss {
				fmt.Printf("  /%s: %d\n", kv.Key, kv.Value)
			}
		}

		printSortedMasks("IPv4 Masks:", masks[0])
		printSortedMasks("IPv6 Masks:", masks[1])

	case "5":
		roas, err := client.GetROAs()
		if err != nil {
			return err
		}
		fmt.Printf("IPv4: Valid: %d, Invalid: %d, Unknown: %d\n", roas.V4v, roas.V4i, roas.V4u)
		fmt.Printf("IPv6: Valid: %d, Invalid: %d, Unknown: %d\n", roas.V6v, roas.V6i, roas.V6u)

	case "6":
		large, err := client.GetLargeCommunities()
		if err != nil {
			return err
		}
		fmt.Printf("IPv4 with Large Communities: %d\n", large.V4)
		fmt.Printf("IPv6 with Large Communities: %d\n", large.V6)

	case "7":
		fmt.Print("Enter Source ASN (e.g. 15169): ")
		if !scanner.Scan() {
			return nil
		}
		asnStr := scanner.Text()
		asn, err := strconv.ParseUint(asnStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ASN: %v", err)
		}
		prefixes, err := client.GetIPv4FromSource(uint32(asn))
		if err != nil {
			return err
		}
		fmt.Printf("Found %d IPv4 prefixes for AS%d:\n", len(prefixes), asn)
		for i, p := range prefixes {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(prefixes)-10)
				break
			}
			fmt.Println("  " + p.String())
		}

	case "8":
		fmt.Print("Enter Source ASN (e.g. 15169): ")
		if !scanner.Scan() {
			return nil
		}
		asnStr := scanner.Text()
		asn, err := strconv.ParseUint(asnStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ASN: %v", err)
		}
		prefixes, err := client.GetIPv6FromSource(uint32(asn))
		if err != nil {
			return err
		}
		fmt.Printf("Found %d IPv6 prefixes for AS%d:\n", len(prefixes), asn)
		for i, p := range prefixes {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(prefixes)-10)
				break
			}
			fmt.Println("  " + p.String())
		}

	case "9":
		fmt.Print("Enter IP Address: ")
		if !scanner.Scan() {
			return nil
		}
		ipStr := scanner.Text()
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address")
		}
		asn, found, err := client.GetOriginFromIP(ip)
		if err != nil {
			return err
		}
		if !found {
			fmt.Println("Origin not found")
		} else {
			fmt.Printf("Origin ASN: %d\n", asn)
		}

	case "10":
		fmt.Print("Enter IP Address: ")
		if !scanner.Scan() {
			return nil
		}
		ipStr := scanner.Text()
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address")
		}
		path, found, err := client.GetASPathFromIP(ip)
		if err != nil {
			return err
		}
		if !found {
			fmt.Println("AS Path not found")
		} else {
			fmt.Printf("AS Path: %v\n", path.Path)
			if len(path.Set) > 0 {
				fmt.Printf("AS Set: %v\n", path.Set)
			}
		}

	case "11":
		fmt.Print("Enter IP Address: ")
		if !scanner.Scan() {
			return nil
		}
		ipStr := scanner.Text()
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address")
		}
		route, found, err := client.GetRoute(ip)
		if err != nil {
			return err
		}
		if !found {
			fmt.Println("Route not found")
		} else {
			fmt.Printf("Route: %s\n", route.String())
		}

	case "12":
		fmt.Print("Enter IP Prefix (e.g. 1.1.1.0/24): ")
		if !scanner.Scan() {
			return nil
		}
		prefixStr := scanner.Text()
		_, prefix, err := net.ParseCIDR(prefixStr)
		if err != nil {
			return fmt.Errorf("invalid prefix: %v", err)
		}

		fmt.Print("Enter Origin ASN: ")
		if !scanner.Scan() {
			return nil
		}
		asnStr := scanner.Text()
		asn, err := strconv.ParseUint(asnStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ASN: %v", err)
		}

		status, found, err := client.GetROA(prefix, uint32(asn))
		if err != nil {
			return err
		}
		if !found {
			fmt.Println("ROA status not found (Unknown)")
		} else {
			statusStr := "Unknown"
			switch status {
			case clidecode.RValid:
				statusStr = "Valid"
			case clidecode.RInvalid:
				statusStr = "Invalid"
			}
			fmt.Printf("ROA Status: %s\n", statusStr)
		}

	case "13":
		fmt.Print("Enter ASN: ")
		if !scanner.Scan() {
			return nil
		}
		asnStr := scanner.Text()
		asn, err := strconv.ParseUint(asnStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid ASN: %v", err)
		}
		vrps, err := client.GetVRPs(uint32(asn))
		if err != nil {
			return err
		}
		fmt.Printf("Found %d VRPs for AS%d:\n", len(vrps), asn)
		for i, v := range vrps {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(vrps)-10)
				break
			}
			fmt.Printf("  %s MaxLen: %d\n", v.Prefix.String(), v.Max)
		}

	case "14":
		invalids, err := client.GetInvalids()
		if err != nil {
			return err
		}
		fmt.Printf("Found %d ASNs with invalid prefixes\n", len(invalids))
		count := 0
		for asn, prefixes := range invalids {
			if count >= 5 {
				fmt.Printf("... and %d more ASNs\n", len(invalids)-5)
				break
			}
			fmt.Printf("  AS%s: %d invalid prefixes\n", asn, len(prefixes))
			count++
		}

	case "99":
		fmt.Print("Enter Command: ")
		if !scanner.Scan() {
			return nil
		}
		cmd := scanner.Text()
		out, err := client.RunCommand(cmd)
		if err != nil {
			return err
		}
		fmt.Println(out)

	default:
		fmt.Println("Invalid choice")
	}

	return nil
}
