package wasm

import (
	"fmt"
	"net/url"
	"strings"
)

const permNetworkPrefix = "network:"

func validateManifestPermissions(perms []string) error {
	for _, p := range perms {
		if p == "" {
			return fmt.Errorf("empty permission")
		}
		if strings.HasPrefix(p, permNetworkPrefix) {
			target := strings.TrimPrefix(p, permNetworkPrefix)
			if target == "" {
				return fmt.Errorf("network permission requires URL prefix")
			}
			if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
				return fmt.Errorf("network permission must start with http:// or https://")
			}
		}
	}
	return nil
}

func allowsNetwork(perms []string, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	normalized := u.Scheme + "://" + u.Host
	if u.Scheme == "https" && strings.HasSuffix(u.Host, ":443") {
		normalized = "https://" + strings.TrimSuffix(u.Host, ":443")
	}
	for _, p := range perms {
		if !strings.HasPrefix(p, permNetworkPrefix) {
			continue
		}
		allowed := strings.TrimPrefix(p, permNetworkPrefix)
		if strings.HasPrefix(rawURL, allowed) || strings.HasPrefix(normalized, allowed) {
			return true
		}
	}
	return false
}
