package routing

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	TypeBuiltin = "builtin"
	TypeRemote  = "remote"

	BuiltinSMTP  = "smtp_email"
	BuiltinInApp = "inapp"
)

// File is the on-disk Traefik-style providers.yaml root.
type File struct {
	Services  map[string]ServiceConfig  `yaml:"services"`
	Providers map[string]ProviderConfig `yaml:"providers"`
}

type ServiceConfig struct {
	Endpoint    string             `yaml:"endpoint"`
	HealthCheck *HealthCheckConfig `yaml:"healthCheck,omitempty"`
}

type HealthCheckConfig struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
}

type ProviderConfig struct {
	Type    string `yaml:"type"`
	Service string `yaml:"service,omitempty"`
	Channel string `yaml:"channel"`
	Timeout string `yaml:"timeout,omitempty"`
}

// Snapshot is an immutable validated routing table.
type Snapshot struct {
	Services  map[string]Service
	Providers map[string]Provider
}

type Service struct {
	Name        string
	Endpoint    string
	HealthCheck *HealthCheck
}

type HealthCheck struct {
	Path     string
	Interval time.Duration
	Timeout  time.Duration
}

type Provider struct {
	Name     string
	Type     string
	Service  string
	Channel  string
	Timeout  time.Duration
	Endpoint string // resolved from service for remote; empty for builtin
}

func LoadFile(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read providers config: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Snapshot, error) {
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse providers yaml: %w", err)
	}
	return Validate(&f)
}

func Validate(f *File) (*Snapshot, error) {
	if f == nil {
		return nil, fmt.Errorf("empty providers config")
	}
	if len(f.Providers) == 0 {
		return nil, fmt.Errorf("providers: at least one provider required")
	}

	services := make(map[string]Service, len(f.Services))
	for name, sc := range f.Services {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("services: empty name")
		}
		ep := strings.TrimSpace(sc.Endpoint)
		if ep == "" {
			return nil, fmt.Errorf("services.%s: endpoint required", name)
		}
		u, err := url.Parse(ep)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, fmt.Errorf("services.%s: invalid endpoint %q", name, ep)
		}
		svc := Service{Name: name, Endpoint: ep}
		if sc.HealthCheck != nil {
			hc, err := parseHealthCheck(sc.HealthCheck)
			if err != nil {
				return nil, fmt.Errorf("services.%s.healthCheck: %w", name, err)
			}
			svc.HealthCheck = hc
		}
		services[name] = svc
	}

	providers := make(map[string]Provider, len(f.Providers))
	for name, pc := range f.Providers {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("providers: empty name")
		}
		typ := strings.TrimSpace(strings.ToLower(pc.Type))
		channel := strings.TrimSpace(pc.Channel)
		if channel == "" {
			return nil, fmt.Errorf("providers.%s: channel required", name)
		}
		timeout := 10 * time.Second
		if pc.Timeout != "" {
			d, err := time.ParseDuration(pc.Timeout)
			if err != nil {
				return nil, fmt.Errorf("providers.%s: invalid timeout %q", name, pc.Timeout)
			}
			timeout = d
		}

		p := Provider{Name: name, Type: typ, Channel: channel, Timeout: timeout}
		switch typ {
		case TypeBuiltin:
			if strings.TrimSpace(pc.Service) != "" {
				return nil, fmt.Errorf("providers.%s: builtin must not set service", name)
			}
			if name != BuiltinSMTP && name != BuiltinInApp {
				return nil, fmt.Errorf("providers.%s: builtin name must be %s or %s", name, BuiltinSMTP, BuiltinInApp)
			}
			if name == BuiltinSMTP && channel != "email" {
				return nil, fmt.Errorf("providers.%s: channel must be email", name)
			}
			if name == BuiltinInApp && channel != "inapp" {
				return nil, fmt.Errorf("providers.%s: channel must be inapp", name)
			}
		case TypeRemote:
			svcName := strings.TrimSpace(pc.Service)
			if svcName == "" {
				return nil, fmt.Errorf("providers.%s: remote requires service", name)
			}
			svc, ok := services[svcName]
			if !ok {
				return nil, fmt.Errorf("providers.%s: service %q not found", name, svcName)
			}
			p.Service = svcName
			p.Endpoint = svc.Endpoint
		default:
			return nil, fmt.Errorf("providers.%s: type must be builtin or remote", name)
		}
		providers[name] = p
	}

	return &Snapshot{Services: services, Providers: providers}, nil
}

func parseHealthCheck(hc *HealthCheckConfig) (*HealthCheck, error) {
	path := strings.TrimSpace(hc.Path)
	if path == "" {
		path = "/health"
	}
	interval := 5 * time.Second
	if hc.Interval != "" {
		d, err := time.ParseDuration(hc.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval %q", hc.Interval)
		}
		interval = d
	}
	timeout := 2 * time.Second
	if hc.Timeout != "" {
		d, err := time.ParseDuration(hc.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q", hc.Timeout)
		}
		timeout = d
	}
	return &HealthCheck{Path: path, Interval: interval, Timeout: timeout}, nil
}
