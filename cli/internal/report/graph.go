package report

import (
	"fmt"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
)

// GenerateGraph produces a Mermaid flowchart from session events and service interactions.
func GenerateGraph(detail *api.SessionDetail, services []ServiceInteraction) string {
	var b strings.Builder

	b.WriteString("graph TD\n")
	b.WriteString("    classDef l1 fill:#00ccff,stroke:#333,color:#000\n")
	b.WriteString("    classDef l2 fill:#cc33ff,stroke:#333,color:#000\n")
	b.WriteString("    classDef l3 fill:#ffaa00,stroke:#333,color:#000\n")
	b.WriteString("    classDef l4 fill:#ff3366,stroke:#333,color:#000\n")
	b.WriteString("\n")

	if detail == nil || len(detail.Events) == 0 {
		b.WriteString("    A[\"No events\"]\n")
		return b.String()
	}

	type node struct {
		id    string
		label string
		layer int
	}

	var nodes []node
	var edges []string
	nodeID := 'A'

	nextID := func() string {
		if nodeID <= 'Z' {
			s := string(nodeID)
			nodeID++
			return s
		}
		// Overflow to AA, AB, etc.
		overflow := int(nodeID - 'Z' - 1)
		s := fmt.Sprintf("%c%c", 'A'+overflow/26, 'A'+overflow%26)
		nodeID++
		return s
	}

	// Track state for deduplication
	var srcIP string
	var authUser, authPass string
	commandCount := 0
	lastEventType := ""

	flushCommands := func() {
		if commandCount > 0 {
			id := nextID()
			label := fmt.Sprintf("Enumeration\\n%d commands", commandCount)
			nodes = append(nodes, node{id, label, 2})
			commandCount = 0
		}
	}

	for _, ev := range detail.Events {
		switch ev.Event {
		case "connection":
			if srcIP == "" {
				if ip, ok := ev.Data["src_ip"].(string); ok {
					srcIP = ip
				}
				flushCommands()
				id := nextID()
				label := fmt.Sprintf("SSH Connection\\n%s", srcIP)
				nodes = append(nodes, node{id, label, 1})
			}

		case "auth":
			if u, ok := ev.Data["username"].(string); ok {
				authUser = u
			}
			if p, ok := ev.Data["password"].(string); ok {
				authPass = p
			}
			if lastEventType != "auth" {
				flushCommands()
				id := nextID()
				label := fmt.Sprintf("Auth: %s\\n%s", authUser, authPass)
				nodes = append(nodes, node{id, label, 1})
			}

		case "container_spawned":
			depth := 1
			if d, ok := ev.Data["depth"].(float64); ok {
				depth = int(d)
			}
			flushCommands()
			id := nextID()
			label := fmt.Sprintf("Container Spawned\\nDepth %d", depth)
			nodes = append(nodes, node{id, label, 2})

		case "depth_increase":
			newDepth := 0
			if d, ok := ev.Data["new_depth"].(float64); ok {
				newDepth = int(d)
			}
			flushCommands()
			id := nextID()
			label := fmt.Sprintf("Depth Increase\\nDepth %d", newDepth)
			nodes = append(nodes, node{id, label, 2})

		case "blindfold_activated":
			flushCommands()
			id := nextID()
			nodes = append(nodes, node{id, "BLINDFOLD\\nActivated", 3})

		case "command":
			commandCount++

		case "service_connection", "service_auth", "service_query":
			// Handled via services below

		case "api_intercepted":
			if lastEventType != "api_intercepted" {
				flushCommands()
				id := nextID()
				nodes = append(nodes, node{id, "API Intercepted\\nMITM Active", 4})
			}
		}
		lastEventType = ev.Event
	}

	flushCommands()

	// Add service nodes
	serviceNodeIDs := []string{}
	for _, svc := range services {
		if svc.Connections == 0 && svc.Queries == 0 {
			continue
		}
		id := nextID()
		detail := ""
		if svc.Queries > 0 {
			detail = fmt.Sprintf("\\n%d queries", svc.Queries)
		} else if svc.AuthAttempts > 0 {
			detail = fmt.Sprintf("\\n%d auth attempts", svc.AuthAttempts)
		}
		label := fmt.Sprintf("%s :%d%s", svc.Protocol, svc.Port, detail)
		nodes = append(nodes, node{id, label, 4})
		serviceNodeIDs = append(serviceNodeIDs, id)
	}

	// Write nodes
	for _, n := range nodes {
		shape := fmt.Sprintf("[\"%s\"]", n.label)
		if n == nodes[0] {
			shape = fmt.Sprintf("[/\"%s\"/]", n.label) // parallelogram for entry
		}
		fmt.Fprintf(&b, "    %s%s:::", n.id, shape)
		switch n.layer {
		case 1:
			b.WriteString("l1\n")
		case 2:
			b.WriteString("l2\n")
		case 3:
			b.WriteString("l3\n")
		case 4:
			b.WriteString("l4\n")
		default:
			b.WriteString("l1\n")
		}
	}

	b.WriteString("\n")

	// Write sequential edges for non-service nodes
	nonServiceNodes := []node{}
	for _, n := range nodes {
		isService := false
		for _, sid := range serviceNodeIDs {
			if n.id == sid {
				isService = true
				break
			}
		}
		if !isService {
			nonServiceNodes = append(nonServiceNodes, n)
		}
	}

	for i := 0; i < len(nonServiceNodes)-1; i++ {
		edges = append(edges, fmt.Sprintf("    %s --> %s", nonServiceNodes[i].id, nonServiceNodes[i+1].id))
	}

	// Fan-out from last non-service node to all service nodes
	if len(serviceNodeIDs) > 0 && len(nonServiceNodes) > 0 {
		last := nonServiceNodes[len(nonServiceNodes)-1].id
		if len(serviceNodeIDs) == 1 {
			edges = append(edges, fmt.Sprintf("    %s --> %s", last, serviceNodeIDs[0]))
		} else {
			targets := strings.Join(serviceNodeIDs, " & ")
			edges = append(edges, fmt.Sprintf("    %s --> %s", last, targets))
		}
	}

	for _, e := range edges {
		b.WriteString(e)
		b.WriteString("\n")
	}

	return b.String()
}
