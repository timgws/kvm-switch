package config

import (
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"net"
	"os"
	"strings"
)

type Config struct {
	Server  ServerConfig   `toml:"server"`
	TLS     *TlsConfig     `toml:"tls"`
	Drivers []DriverConfig `toml:"drivers"`
	Layout  *LayoutConfig  `toml:"layout"`
}

type ServerConfig struct {
	Addr         string   `toml:"addr"`
	AuthToken    string   `toml:"auth_token"`
	Debug        bool     `toml:"debug"`
	DebugModules []string `toml:"debug_modules"`
}

type TlsConfig struct {
	Enabled         bool     `toml:"enabled"`
	Dir             string   `toml:"dir"`
	CertFile        string   `toml:"cert_file"`
	KeyFile         string   `toml:"key_file"`
	CaFile          string   `toml:"ca_file"`
	DNSNames        []string `toml:"dns_names"`
	IPAddresses     []string `toml:"ip_addresses"`
	RealIPAddresses []net.IP
}

type DriverConfig struct {
	// "startech_kvm" | "blustream"
	Type string `toml:"type"`

	// The short name is what your layout references (e.g. "kvm", "matrix")
	ShortName string `toml:"short_name"`

	// Optional override for long driver name shown in status/UI
	Name string `toml:"name"`

	// Either set SerialDevice explicitly OR enable AutoDiscover.
	SerialDevice string `toml:"serial_device"`
	Baud         int    `toml:"baud"`
	AutoDiscover bool   `toml:"auto_discover"`

	// USB match (best-effort; depends on OS & enumerator support)
	USBVID          uint16 `toml:"usb_vid"`
	USBPID          uint16 `toml:"usb_pid"`
	SerialNumber    string `toml:"serial_number"`
	ProductContains string `toml:"product_contains"`
}

type LayoutConfig struct {
	Computers []ComputerConfig `toml:"computers"`
}

type ComputerConfig struct {
	Name   string         `toml:"name"`
	Left   []ActionConfig `toml:"left"`
	Right  []ActionConfig `toml:"right"`
	Top    []ActionConfig `toml:"top"`
	Bottom []ActionConfig `toml:"bottom"`
}

type ActionConfig struct {
	Driver string `toml:"driver"` // driver short_name
	Action string `toml:"action"` // e.g. "01-03" or "4"
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = "config.toml"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8787"
	}

	if len(c.Drivers) == 0 {
		c.Drivers = []DriverConfig{
			{
				Type: "startech_kvm", ShortName: "kvm",
				SerialDevice: "/dev/tty.usbserial-141140", Baud: 115200,
			},
			{
				Type: "blustream", ShortName: "matrix",
				SerialDevice: "/dev/tty.usbserial-141130", Baud: 57600,
			},
		}
	}

	for i := range c.Drivers {
		if c.Drivers[i].ShortName == "" {
			switch c.Drivers[i].Type {
			case "startech_kvm":
				c.Drivers[i].ShortName = "kvm"
			case "blustream":
				c.Drivers[i].ShortName = "matrix"
			}
		}
		if c.Drivers[i].Baud == 0 {
			switch c.Drivers[i].Type {
			case "startech_kvm":
				c.Drivers[i].Baud = 115200
			case "blustream":
				c.Drivers[i].Baud = 57600
			}
		}
	}

	fmt.Printf("==>\n")
	fmt.Printf("==> %v", c.TLS)
	c.TLS.RealIPAddresses, _ = parseIPs(c.TLS.IPAddresses)
	fmt.Printf("==>\n")
}

func parseIPs(ipStrs []string) ([]net.IP, error) {
	ips := make([]net.IP, 0, len(ipStrs))
	for _, s := range ipStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP in config: %q", s)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

func (c *Config) validate() error {
	if c.Server.Addr == "" {
		return errors.New("server.addr is required")
	}

	shortNames := map[string]struct{}{}
	for i, d := range c.Drivers {
		if d.Type == "" {
			return fmt.Errorf("drivers[%d].type is required", i)
		}
		if d.ShortName == "" {
			return fmt.Errorf("drivers[%d].short_name is required", i)
		}
		if _, ok := shortNames[d.ShortName]; ok {
			return fmt.Errorf("duplicate drivers short_name: %q", d.ShortName)
		}
		shortNames[d.ShortName] = struct{}{}

		// allow SerialDevice empty if AutoDiscover true
		if d.SerialDevice == "" && !d.AutoDiscover {
			// not fatal, but will fail at runtime; better to fail fast:
			return fmt.Errorf("drivers[%d] has no serial_device and auto_discover=false", i)
		}
	}

	if c.Layout != nil {
		for i, comp := range c.Layout.Computers {
			if comp.Name == "" {
				return fmt.Errorf("layout.computers[%d].name is required", i)
			}
			all := append(append(append(comp.Left, comp.Right...), comp.Top...), comp.Bottom...)
			for _, a := range all {
				if a.Driver == "" || a.Action == "" {
					return fmt.Errorf("layout action missing driver/action (computer=%q)", comp.Name)
				}
				if _, ok := shortNames[a.Driver]; !ok {
					return fmt.Errorf("layout references unknown driver short_name %q (computer=%q)", a.Driver, comp.Name)
				}
			}
		}
	}

	return nil
}
