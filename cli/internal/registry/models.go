package registry

// EnvironmentPorts holds the host port assignments for a deployment.
type EnvironmentPorts struct {
	SSH       int `json:"ssh,omitempty"`
	HTTP      int `json:"http,omitempty"`
	Dashboard int `json:"dashboard,omitempty"`
}

// Environment represents a tracked LABYRINTH deployment.
// JSON schema is backwards-compatible with deploy.sh.
type Environment struct {
	Name           string           `json:"name"`
	Type           string           `json:"type"`                      // "test" or "production"
	Mode           string           `json:"mode"`                      // "docker-compose", "docker", "k8s", "edge"
	Created        string           `json:"created"`                   // RFC3339 timestamp
	ComposeProject string           `json:"compose_project,omitempty"` // For docker-compose/docker mode
	Namespace      string           `json:"namespace,omitempty"`       // For k8s mode
	Ports          EnvironmentPorts `json:"ports,omitempty"`           // Assigned host ports
	Subnet         string           `json:"subnet,omitempty"`          // Docker network subnet
	DashboardURL   string           `json:"dashboard_url,omitempty"`   // Dashboard URL
	ConfigPath     string           `json:"config_path,omitempty"`     // Path to labyrinth.yaml
	ComposeFile    string           `json:"compose_file,omitempty"`    // Path to generated docker-compose.yml
}
