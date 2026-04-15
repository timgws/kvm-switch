package serialdiscovery

import (
	"fmt"
	"strconv"
	"strings"

	"go.bug.st/serial/enumerator"
)

type Match struct {
	VID uint16
	PID uint16

	SerialNumber    string
	ProductContains string
}

func Resolve(explicit string, auto bool, m Match) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if !auto {
		return "", fmt.Errorf("serial device not configured and auto_discover=false")
	}
	return Discover(m)
}

func Discover(m Match) (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", err
	}

	var matches []string
	for _, p := range ports {
		if p == nil || p.Name == "" {
			continue
		}

		// If user asked for USB matching, require USB.
		if (m.VID != 0 || m.PID != 0 || m.SerialNumber != "" || m.ProductContains != "") && !p.IsUSB {
			continue
		}

		if p.IsUSB {
			if m.VID != 0 {
				vid, ok := parseHexU16(p.VID)
				if !ok || vid != m.VID {
					continue
				}
			}
			if m.PID != 0 {
				pid, ok := parseHexU16(p.PID)
				if !ok || pid != m.PID {
					continue
				}
			}

			if m.SerialNumber != "" && !strings.EqualFold(p.SerialNumber, m.SerialNumber) {
				continue
			}
			if m.ProductContains != "" && !containsFold(p.Product, m.ProductContains) {
				continue
			}
		}

		matches = append(matches, p.Name)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf(
			"no serial ports matched (VID=%#04x PID=%#04x Serial=%q Product~%q)",
			m.VID, m.PID, m.SerialNumber, m.ProductContains,
		)
	}

	// Deterministic choice: first match. If you get >1, tighten the filter (serial_number/product_contains/etc).
	return matches[0], nil
}

// parseHexU16 parses "0403" / "6001" (and tolerates "0x0403") into uint16.
func parseHexU16(s string) (uint16, bool) {
	x := strings.TrimSpace(strings.ToLower(s))
	x = strings.TrimPrefix(x, "0x")
	if x == "" {
		return 0, false
	}
	u, err := strconv.ParseUint(x, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(u), true
}

func containsFold(hay, needle string) bool {
	return strings.Contains(strings.ToLower(hay), strings.ToLower(needle))
}
