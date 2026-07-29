package application

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// KubernetesMinorVersion returns the major.minor component consumed by the
// Ansible inventory and package-selection variables. Keeping this derivation
// in Provisioning prevents transport adapters from owning deployment policy.
func KubernetesMinorVersion(version string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// InventoryHost is the transport-neutral host description used to render an
// Ansible inventory. The runtime adapter owns credential lookup and temporary
// private-key files; this policy deliberately only decides its YAML shape.
type InventoryHost struct {
	Alias    string
	IP       string
	SSHPort  int
	User     string
	AuthType string
	Password string
	KeyFile  string
}

func InventoryNodeAlias(role string, index int, ip string) string {
	prefix := "worker"
	if strings.EqualFold(strings.TrimSpace(role), "master") {
		prefix = "master"
	}
	return fmt.Sprintf("%s%02d-%s", prefix, index, strings.ReplaceAll(strings.TrimSpace(ip), ".", "-"))
}

// MarshalAnsibleInventory produces the YAML inventory consumed by the
// deployment playbook. When maskSecret is true, it never exposes passwords or
// local private-key paths in API responses and logs.
func MarshalAnsibleInventory(masters, workers []InventoryHost, maskSecret bool) (string, error) {
	hosts := func(items []InventoryHost) map[string]any {
		result := make(map[string]any, len(items))
		for _, host := range items {
			variables := map[string]any{
				"ansible_host": strings.TrimSpace(host.IP),
				"ansible_port": host.SSHPort,
				"ansible_user": strings.TrimSpace(host.User),
			}
			if strings.TrimSpace(host.KeyFile) != "" {
				keyFile := host.KeyFile
				if maskSecret {
					keyFile = "<temporary-private-key>"
				}
				variables["ansible_ssh_private_key_file"] = keyFile
			} else {
				password := host.Password
				if maskSecret {
					password = "***"
				}
				variables["ansible_password"] = password
				variables["ansible_become_password"] = password
			}
			alias := strings.TrimSpace(host.Alias)
			if alias == "" {
				alias = strings.TrimSpace(host.IP)
			}
			result[alias] = variables
		}
		return result
	}

	document := map[string]any{
		"all": map[string]any{
			"vars": map[string]any{"ansible_python_interpreter": "/usr/bin/python3"},
			"children": map[string]any{
				"master": map[string]any{"hosts": hosts(masters)},
				"worker": map[string]any{"hosts": hosts(workers)},
			},
		},
	}
	data, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
