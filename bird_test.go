package clidecode

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

// mockQuerier returns a function that simulates BIRD socket responses
func mockQuerier(responses map[string]string) func(string, string) (string, error) {
	return func(socketPath, command string) (string, error) {
		if resp, ok := responses[command]; ok {
			return resp, nil
		}
		return "", fmt.Errorf("unexpected command: %s", command)
	}
}

func TestGetBGPTotal(t *testing.T) {
	responses := map[string]string{
		"show route count": `1007-2076414 of 2076414 routes for 1038207 networks in table master4
 471160 of 471160 routes for 235580 networks in table master6
 Total: 2547574 of 2547574 routes for 1273787 networks in 2 tables`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	totals, err := client.GetBGPTotal()
	if err != nil {
		t.Fatalf("GetBGPTotal failed: %v", err)
	}

	expected := Totals{
		V4Rib: 2076414,
		V4Fib: 1038207,
		V6Rib: 471160,
		V6Fib: 235580,
	}

	if totals != expected {
		t.Errorf("Expected %+v, got %+v", expected, totals)
	}
}

func TestGetPeers(t *testing.T) {
	responses := map[string]string{
		"show protocols": `BIRD 2.0.8 ready.
name     proto    table    state  since       info
bgp1_v4  BGP      master4  up     10:00:00    Established   
  BGP state:          Established
  Neighbor address:   192.0.2.1
  Neighbor AS:        65001
bgp2_v4  BGP      master4  start  10:00:00    Connect       
  BGP state:          Connect
  Neighbor address:   192.0.2.2
  Neighbor AS:        65002
bgp3_v6  BGP      master6  up     10:00:00    Established   
  BGP state:          Established
  Neighbor address:   2001:db8::1
  Neighbor AS:        65003`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	peers, err := client.GetPeers()
	if err != nil {
		t.Fatalf("GetPeers failed: %v", err)
	}

	expected := Peers{
		V4c: 2, V4e: 1,
		V6c: 1, V6e: 1,
	}

	if peers != expected {
		t.Errorf("Expected %+v, got %+v", expected, peers)
	}
}

func TestGetRoute(t *testing.T) {
	responses := map[string]string{
		"show route primary for 8.8.8.8": `Table master4:
8.8.8.0/24           unreachable [BGP3v4 2025-11-19 from 192.110.255.57] * (100) [AS15169i]
	Type: BGP univ
	BGP.origin: IGP`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	ip := net.ParseIP("8.8.8.8")
	route, found, err := client.GetRoute(ip)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}
	if !found {
		t.Fatal("Route not found")
	}

	if route.String() != "8.8.8.0/24" {
		t.Errorf("Expected 8.8.8.0/24, got %s", route.String())
	}
}

func TestGetRoute_NotFound(t *testing.T) {
	responses := map[string]string{
		"show route primary for 1.2.3.4": `Network not in table`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	ip := net.ParseIP("1.2.3.4")
	_, found, err := client.GetRoute(ip)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}
	if found {
		t.Error("Expected route to not be found")
	}
}

func TestGetVersion(t *testing.T) {
	responses := map[string]string{
		"show status": `BIRD 2.0.8 ready.
Router ID is 192.168.1.1`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	ver, err := client.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if ver != "BIRD 2.0.8 ready." {
		t.Errorf("Expected 'BIRD 2.0.8 ready.', got '%s'", ver)
	}
}

func TestGetMasks(t *testing.T) {
	responses := map[string]string{
		"show route primary table master4": `1.0.0.0/24 via 192.0.2.1
2.0.0.0/24 via 192.0.2.1
3.0.0.0/16 via 192.0.2.1`,
		"show route primary table master6": `2001:db8::/32 via fe80::1
2001:db8:1::/48 via fe80::1`,
	}

	client := &BirdClient{
		Querier: mockQuerier(responses),
	}

	masks, err := client.GetMasks()
	if err != nil {
		t.Fatalf("GetMasks failed: %v", err)
	}

	expectedV4 := map[string]uint32{"24": 2, "16": 1}
	expectedV6 := map[string]uint32{"32": 1, "48": 1}

	if !reflect.DeepEqual(masks[0], expectedV4) {
		t.Errorf("IPv4 masks mismatch. Got %v, want %v", masks[0], expectedV4)
	}
	if !reflect.DeepEqual(masks[1], expectedV6) {
		t.Errorf("IPv6 masks mismatch. Got %v, want %v", masks[1], expectedV6)
	}
}

func TestDecodeASPaths(t *testing.T) {
	tests := []struct {
		Name     string
		path     string
		wantPath []uint32
		wantSet  []uint32
	}{
		{
			Name:     "Single AS",
			path:     "3356 12345",
			wantPath: []uint32{3356, 12345},
		},
		{
			Name:     "Dual AS",
			path:     "3356 12345 9876",
			wantPath: []uint32{3356, 12345, 9876},
		},
		{
			Name:     "Single AS-SET",
			path:     "3356 12345 9876 {1212}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212},
		},
		{
			Name:     "Dual AS-SET",
			path:     "3356 12345 9876 {1212 3434}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212, 3434},
		},
	}

	for _, tc := range tests {
		gotPath, gotSet := decodeASPaths(tc.path)
		if !reflect.DeepEqual(gotPath, tc.wantPath) {
			t.Errorf("Got %v, Wanted %v", gotPath, tc.wantPath)
		}
		if !reflect.DeepEqual(gotSet, tc.wantSet) {
			t.Errorf("Got %v, Wanted %v", gotSet, tc.wantSet)
		}
	}
}

func BenchmarkDecodeASPaths(b *testing.B) {
	tests := []struct {
		Name     string
		path     string
		wantPath []uint32
		wantSet  []uint32
	}{
		{
			Name:     "Single AS",
			path:     "3356 12345",
			wantPath: []uint32{3356, 12345},
		},
		{
			Name:     "Dual AS",
			path:     "3356 12345 9876",
			wantPath: []uint32{3356, 12345, 9876},
		},
		{
			Name:     "Single AS-SET",
			path:     "3356 12345 9876 {1212}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212},
		},
		{
			Name:     "Dual AS-SET",
			path:     "3356 12345 9876 {1212 3434}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212, 3434},
		},
	}
	for _, tc := range tests {
		for n := 0; n < b.N; n++ {
			decodeASPaths(tc.path)
		}
	}
}
