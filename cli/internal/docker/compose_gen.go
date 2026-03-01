package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeGenConfig holds parameters for generating a production compose file.
type ComposeGenConfig struct {
	EnvName      string
	Ports        PortSet
	Subnet       string
	ProxyIP      string
	BaseCompose  string // path to base docker-compose.yml
	OutputDir    string // directory to write generated file
	BuildContext string // absolute path to repo root for build contexts
}

// GenerateComposeFile reads the base docker-compose.yml and produces a
// remapped version for a production environment with unique names/ports/subnet.
func GenerateComposeFile(cfg ComposeGenConfig) (string, error) {
	data, err := os.ReadFile(cfg.BaseCompose)
	if err != nil {
		return "", fmt.Errorf("read base compose: %w", err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return "", fmt.Errorf("parse base compose: %w", err)
	}

	// Remap services
	if services, ok := compose["services"].(map[string]interface{}); ok {
		remapServices(services, cfg)
	}

	// Remap networks
	remapNetworks(compose, cfg)

	// Remap volumes
	remapVolumes(compose, cfg)

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	outPath := filepath.Join(cfg.OutputDir, "docker-compose.yml")
	outData, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("marshal compose: %w", err)
	}

	header := fmt.Sprintf("# LABYRINTH — Production Compose (%s)\n# Auto-generated — do not edit manually\n\n", cfg.EnvName)
	if err := os.WriteFile(outPath, []byte(header+string(outData)), 0644); err != nil {
		return "", fmt.Errorf("write compose: %w", err)
	}

	return outPath, nil
}

func remapServices(services map[string]interface{}, cfg ComposeGenConfig) {
	prefix := cfg.EnvName

	portMap := map[string]string{
		"22:22":      fmt.Sprintf("%d:22", cfg.Ports.SSH),
		"8080:80":    fmt.Sprintf("%d:80", cfg.Ports.HTTP),
		"9000:9000":  fmt.Sprintf("%d:9000", cfg.Ports.Dashboard),
	}

	for svcName, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Remap container_name
		if name, ok := svc["container_name"].(string); ok {
			svc["container_name"] = strings.Replace(name, "labyrinth-", prefix+"-", 1)
		}

		// Remap image name if present
		if img, ok := svc["image"].(string); ok {
			svc["image"] = strings.Replace(img, "labyrinth-", prefix+"-", 1)
		}

		// Remap build context to absolute path
		if build, ok := svc["build"].(map[string]interface{}); ok {
			if ctx, ok := build["context"].(string); ok && ctx == "." {
				build["context"] = cfg.BuildContext
			}
		}

		// Remap ports
		if ports, ok := svc["ports"].([]interface{}); ok {
			for i, p := range ports {
				if ps, ok := p.(string); ok {
					if mapped, exists := portMap[ps]; exists {
						ports[i] = mapped
					}
				}
			}
		}

		// Remap network static IP for proxy
		if nets, ok := svc["networks"].(map[string]interface{}); ok {
			for netName, netCfg := range nets {
				if netCfgMap, ok := netCfg.(map[string]interface{}); ok {
					if _, ok := netCfgMap["ipv4_address"]; ok {
						netCfgMap["ipv4_address"] = cfg.ProxyIP
					}
				}
				// Rename network reference
				if netName == "labyrinth-net" {
					nets[prefix+"-net"] = nets[netName]
					delete(nets, netName)
				}
			}
		}
		// Handle simple network references (list form)
		if nets, ok := svc["networks"].([]interface{}); ok {
			for i, n := range nets {
				if ns, ok := n.(string); ok && ns == "labyrinth-net" {
					nets[i] = prefix + "-net"
				}
			}
		}

		// Inject environment variables
		envVars := []string{
			fmt.Sprintf("LABYRINTH_MODE=production"),
			fmt.Sprintf("LABYRINTH_ENV_NAME=%s", cfg.EnvName),
			fmt.Sprintf("LABYRINTH_ENV_TYPE=production"),
		}

		// Only add env vars to services that already have environment
		if existing, ok := svc["environment"].([]interface{}); ok {
			// Filter out any existing LABYRINTH_MODE/ENV_NAME/ENV_TYPE
			var filtered []interface{}
			for _, e := range existing {
				if es, ok := e.(string); ok {
					if strings.HasPrefix(es, "LABYRINTH_MODE=") ||
						strings.HasPrefix(es, "LABYRINTH_ENV_NAME=") ||
						strings.HasPrefix(es, "LABYRINTH_ENV_TYPE=") {
						continue
					}
				}
				filtered = append(filtered, e)
			}
			for _, ev := range envVars {
				filtered = append(filtered, ev)
			}
			svc["environment"] = filtered
		} else if svcName != "session-template" {
			// Add environment block to services that don't have one (except session-template)
			var envList []interface{}
			for _, ev := range envVars {
				envList = append(envList, ev)
			}
			svc["environment"] = envList
		}

		// Remap volume references
		if vols, ok := svc["volumes"].([]interface{}); ok {
			for i, v := range vols {
				if vs, ok := v.(string); ok {
					vols[i] = strings.Replace(vs, "forensic-data:", prefix+"-forensic-data:", 1)
				}
			}
		}

		// Remap depends_on
		// (keep service names as-is since compose uses service keys, not container names)
	}
}

func remapNetworks(compose map[string]interface{}, cfg ComposeGenConfig) {
	networks, ok := compose["networks"].(map[string]interface{})
	if !ok {
		return
	}

	prefix := cfg.EnvName
	if netCfg, ok := networks["labyrinth-net"]; ok {
		// Update subnet
		if netMap, ok := netCfg.(map[string]interface{}); ok {
			if ipam, ok := netMap["ipam"].(map[string]interface{}); ok {
				if cfgList, ok := ipam["config"].([]interface{}); ok {
					for _, c := range cfgList {
						if cm, ok := c.(map[string]interface{}); ok {
							cm["subnet"] = cfg.Subnet
						}
					}
				}
			}
		}
		networks[prefix+"-net"] = netCfg
		delete(networks, "labyrinth-net")
	}
}

func remapVolumes(compose map[string]interface{}, cfg ComposeGenConfig) {
	volumes, ok := compose["volumes"].(map[string]interface{})
	if !ok {
		return
	}

	prefix := cfg.EnvName
	if volCfg, ok := volumes["forensic-data"]; ok {
		volumes[prefix+"-forensic-data"] = volCfg
		delete(volumes, "forensic-data")
	}
}
