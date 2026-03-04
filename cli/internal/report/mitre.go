package report

import "strings"

// MapEvent returns the MITRE ATT&CK tactic and technique ID for a forensic event.
func MapEvent(eventType string, data map[string]interface{}) (tactic, techID string) {
	switch eventType {
	case "http_access":
		return "Reconnaissance", "T1595"
	case "connection":
		return "Reconnaissance", "T1046"
	case "auth":
		return "Credential Access", "T1110"
	case "container_spawned":
		return "Initial Access", "T1078"
	case "depth_increase":
		return "Lateral Movement", "T1570"
	case "blindfold_activated":
		return "Defense Evasion", "T1562"
	case "service_connection":
		return "Lateral Movement", "T1021"
	case "service_auth":
		return "Credential Access", "T1078"
	case "service_query":
		return "Collection", "T1005"
	case "api_intercepted":
		return "Collection", "T1557"
	case "command":
		return mapCommand(data)
	}
	return "", ""
}

func mapCommand(data map[string]interface{}) (string, string) {
	cmd, _ := data["command"].(string)
	if cmd == "" {
		return "Execution", ""
	}

	lower := strings.ToLower(cmd)
	parts := strings.Fields(lower)
	if len(parts) == 0 {
		return "Execution", ""
	}

	base := parts[0]

	// Credential access patterns
	if base == "env" || base == "printenv" {
		return "Credential Access", "T1552"
	}
	if base == "cat" || base == "head" || base == "tail" || base == "less" || base == "more" {
		for _, arg := range parts[1:] {
			if strings.Contains(arg, ".env") || strings.Contains(arg, "credential") ||
				strings.Contains(arg, "secret") || strings.Contains(arg, "password") ||
				strings.Contains(arg, "token") || strings.Contains(arg, ".aws") ||
				strings.Contains(arg, "shadow") || strings.Contains(arg, ".ssh") {
				return "Credential Access", "T1552"
			}
		}
	}

	// Lateral movement patterns
	if base == "ssh" || base == "curl" || base == "wget" || base == "psql" ||
		base == "mysql" || base == "redis-cli" || base == "mongo" || base == "nc" ||
		base == "ncat" || base == "socat" {
		return "Lateral Movement", "T1021"
	}

	// Discovery patterns
	if base == "ls" || base == "find" || base == "tree" || base == "du" ||
		base == "stat" || base == "file" || base == "locate" {
		return "Discovery", "T1083"
	}
	if base == "whoami" || base == "id" || base == "groups" || base == "w" || base == "who" {
		return "Discovery", "T1033"
	}
	if base == "ifconfig" || base == "ip" || base == "netstat" || base == "ss" ||
		base == "arp" || base == "route" || base == "nmap" {
		return "Discovery", "T1046"
	}
	if base == "ps" || base == "top" || base == "pgrep" {
		return "Discovery", "T1057"
	}
	if base == "uname" || base == "hostname" || base == "cat" {
		if base == "cat" {
			for _, arg := range parts[1:] {
				if strings.Contains(arg, "/etc/os-release") || strings.Contains(arg, "/etc/hostname") ||
					strings.Contains(arg, "/proc/") {
					return "Discovery", "T1082"
				}
			}
		} else {
			return "Discovery", "T1082"
		}
	}

	return "Execution", ""
}
