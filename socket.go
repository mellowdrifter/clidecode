package clidecode

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// querySocket sends a command to the BIRD control socket and returns the response.
// The BIRD control protocol works as follows:
// 1. Connect to the socket
// 2. Read the greeting (starts with '0001')
// 3. Send the command
// 4. Read response lines until a final status code is received
// Response codes:
//   - 0000-0999: Success codes (0000 = completion, others may have data)
//   - 8000-8999: Runtime errors
//   - 9000-9999: Parse errors
//
// Lines starting with ' ' (space) are continuation lines (part of previous line's data)
// Lines starting with '+' are data lines with code
func querySocket(socketPath, command string) (string, error) {
	// Connect to Unix socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to socket %s: %w", socketPath, err)
	}
	defer conn.Close()

	// Set a deadline for all operations
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReader(conn)

	// Read the greeting
	greeting, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read greeting: %w", err)
	}

	// Check if greeting indicates ready state (code 0001)
	if !strings.HasPrefix(greeting, "0001") {
		return "", fmt.Errorf("unexpected greeting: %s", greeting)
	}

	// Send the command
	// Try sending with \r\n as some servers might be strict
	_, err = fmt.Fprintf(conn, "%s\r\n", command)
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read the response
	var output strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		// Remove trailing newline
		line = strings.TrimRight(line, "\n")

		// Check for continuation line (starts with space)
		if len(line) > 0 && line[0] == ' ' {
			output.WriteString(line[1:]) // Skip the leading space
			output.WriteString("\n")
			continue
		}

		// Check if this is a status line (first 4 characters are digits)
		if len(line) >= 4 {
			code := line[0:4]
			isNumeric := true
			for _, c := range code {
				if c < '0' || c > '9' {
					isNumeric = false
					break
				}
			}

			if isNumeric {
				// Check for final status codes
				// 0000-0999 are success codes
				// 8000-8999 are runtime errors
				// 9000-9999 are parse errors
				if code[0] == '0' {
					// Success - command completed
					// If it's a data line (has space after code), include it
					if len(line) > 5 && (line[4] == ' ' || line[4] == '-') {
						output.WriteString(line[5:])
						output.WriteString("\n")
					}
					break
				} else if code[0] == '8' || code[0] == '9' {
					// Error code
					return "", fmt.Errorf("BIRD error: %s", line)
				} else if code[0] >= '1' && code[0] <= '9' {
					// Other codes (shouldn't happen for final status if we check 0xxx)
					// Data line - extract the message part (skip code and separator)
					if len(line) > 5 {
						// Lines are formatted as "CODE-message" or "CODE message"
						// We skip the first 5 characters (code + separator)
						if line[4] == '-' || line[4] == ' ' {
							output.WriteString(line[5:])
							output.WriteString("\n")
						}
					}
				}
			} else {
				// Not a status code (e.g. route starting with 8.8.8.8)
				// Treat as data line
				output.WriteString(line)
				output.WriteString("\n")
			}
		} else if len(line) > 0 {
			// Short line, treat as data
			output.WriteString(line)
			output.WriteString("\n")
		}
	}

	return strings.TrimSpace(output.String()), nil
}
