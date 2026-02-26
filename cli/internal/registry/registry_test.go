package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterAndLoad(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	env := Environment{
		Name:           "test-env",
		Type:           "test",
		Mode:           "docker-compose",
		Created:        "2026-02-26T10:00:00Z",
		ComposeProject: "labyrinth-test-env",
	}

	if err := reg.Register(env); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	loaded, err := reg.Load("test-env")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Name != env.Name {
		t.Errorf("Name mismatch: got %q, want %q", loaded.Name, env.Name)
	}
	if loaded.Type != env.Type {
		t.Errorf("Type mismatch: got %q, want %q", loaded.Type, env.Type)
	}
	if loaded.Mode != env.Mode {
		t.Errorf("Mode mismatch: got %q, want %q", loaded.Mode, env.Mode)
	}
	if loaded.Created != env.Created {
		t.Errorf("Created mismatch: got %q, want %q", loaded.Created, env.Created)
	}
	if loaded.ComposeProject != env.ComposeProject {
		t.Errorf("ComposeProject mismatch: got %q, want %q", loaded.ComposeProject, env.ComposeProject)
	}
}

func TestRemoveEnv(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	env := Environment{
		Name:    "remove-me",
		Type:    "test",
		Mode:    "docker-compose",
		Created: "2026-02-26T10:00:00Z",
	}
	reg.Register(env)

	if err := reg.Remove("remove-me"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := reg.Load("remove-me")
	if err == nil {
		t.Error("Expected error loading removed env, got nil")
	}

	// File should not exist
	path := filepath.Join(dir, "remove-me.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("File still exists after Remove")
	}
}

func TestListAllEmpty(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	envs, err := reg.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(envs) != 0 {
		t.Errorf("Expected 0 envs, got %d", len(envs))
	}
}

func TestListAllMultiple(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	names := []string{"alpha", "beta", "gamma"}
	for _, name := range names {
		reg.Register(Environment{
			Name:    name,
			Type:    "test",
			Mode:    "docker-compose",
			Created: "2026-02-26T10:00:00Z",
		})
	}

	envs, err := reg.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(envs) != 3 {
		t.Errorf("Expected 3 envs, got %d", len(envs))
	}

	// Verify sorted by name
	for i, name := range names {
		if envs[i].Name != name {
			t.Errorf("Env %d: got %q, want %q", i, envs[i].Name, name)
		}
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	_, err := reg.Load("does-not-exist")
	if err == nil {
		t.Error("Expected error loading nonexistent env, got nil")
	}
}

func TestBackwardsCompatDeploySh(t *testing.T) {
	// deploy.sh writes compact JSON like this
	compactJSON := `{"name":"mylab","type":"test","mode":"docker-compose","created":"2026-02-26T10:00:00Z","compose_project":"labyrinth-mylab"}`

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mylab.json"), []byte(compactJSON), 0644)

	reg := New(dir)
	env, err := reg.Load("mylab")
	if err != nil {
		t.Fatalf("Failed to load deploy.sh-style JSON: %v", err)
	}

	if env.Name != "mylab" {
		t.Errorf("Name: got %q, want %q", env.Name, "mylab")
	}
	if env.Type != "test" {
		t.Errorf("Type: got %q, want %q", env.Type, "test")
	}
	if env.Mode != "docker-compose" {
		t.Errorf("Mode: got %q, want %q", env.Mode, "docker-compose")
	}
	if env.ComposeProject != "labyrinth-mylab" {
		t.Errorf("ComposeProject: got %q, want %q", env.ComposeProject, "labyrinth-mylab")
	}
}

func TestRoundtripJSON(t *testing.T) {
	original := Environment{
		Name:           "roundtrip",
		Type:           "production",
		Mode:           "docker",
		Created:        "2026-02-26T12:00:00Z",
		ComposeProject: "labyrinth-roundtrip",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Environment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Errorf("Roundtrip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestOptionalFields(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		hasCompose bool
		hasNamespace bool
	}{
		{
			name: "docker-compose has compose_project",
			env: Environment{
				Name: "dc", Mode: "docker-compose",
				ComposeProject: "labyrinth-dc",
			},
			hasCompose: true,
		},
		{
			name: "k8s has namespace",
			env: Environment{
				Name: "k8s", Mode: "k8s",
				Namespace: "labyrinth-k8s",
			},
			hasNamespace: true,
		},
		{
			name: "edge has neither",
			env: Environment{
				Name: "edge", Mode: "edge",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.env)
			var raw map[string]interface{}
			json.Unmarshal(data, &raw)

			_, hasCP := raw["compose_project"]
			_, hasNS := raw["namespace"]

			if tt.hasCompose && !hasCP {
				t.Error("Expected compose_project in JSON")
			}
			if !tt.hasCompose && hasCP {
				t.Error("Unexpected compose_project in JSON")
			}
			if tt.hasNamespace && !hasNS {
				t.Error("Expected namespace in JSON")
			}
			if !tt.hasNamespace && hasNS {
				t.Error("Unexpected namespace in JSON")
			}
		})
	}
}
