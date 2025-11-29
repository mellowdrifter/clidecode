# BIRD Socket Connection Test Tool

A simple CLI tool to verify Unix socket connectivity with BIRD routing daemon.

## Purpose

This tool tests the socket-based communication with BIRD by attempting to retrieve IPv4 and IPv6 BGP route totals. It's useful for:
- Verifying socket connectivity on production servers
- Testing BIRD2 vs BIRD3 configurations
- Quick diagnostics of BIRD socket availability

## Building

```bash
go build -o birdtest ./cli
```

## Usage

Simply run the binary:

```bash
sudo ./birdtest
```

The tool will:
1. Automatically detect the BIRD socket path
2. Identify the BIRD version
3. Present an interactive menu to test various commands

### Interactive Menu

```
Available Commands:
 1. GetBGPTotal (RIB/FIB counts)
 2. GetPeers (Peer counts)
 3. GetTotalSourceASNs (Unique ASN counts)
 4. GetMasks (Prefix length distribution)
 5. GetROAs (ROA state counts)
 6. GetLargeCommunities (Large community counts)
 7. GetIPv4FromSource (Prefixes by Origin ASN)
 8. GetIPv6FromSource (Prefixes by Origin ASN)
 9. GetOriginFromIP (Origin ASN for IP)
10. GetASPathFromIP (AS Path for IP)
11. GetRoute (FIB entry for IP)
12. GetROA (ROA status for IP/ASN)
13. GetVRPs (VRPs for ASN)
14. GetInvalids (RPKI Invalid prefixes)
 0. Exit
```

Select an option by entering the number and following the prompts.

## Socket Paths

The tool checks these standard locations:
- `/run/bird.ctl`
- `/run/bird/bird.ctl`
- `/run/bird3.ctl`
- `/var/run/bird.ctl`
- `/var/run/bird3.ctl`
