package registry

// Environment represents a tracked LABYRINTH deployment.
// JSON schema is backwards-compatible with deploy.sh.
type Environment struct {
	Name           string `json:"name"`
	Type           string `json:"type"`                      // "test" or "production"
	Mode           string `json:"mode"`                      // "docker-compose", "docker", "k8s", "edge"
	Created        string `json:"created"`                   // RFC3339 timestamp
	ComposeProject string `json:"compose_project,omitempty"` // For docker-compose/docker mode
	Namespace      string `json:"namespace,omitempty"`       // For k8s mode
}
