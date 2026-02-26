package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Registry manages environment JSON files in the env directory.
type Registry struct {
	dir string
}

// New creates a Registry. If dir is empty, defaults to ~/.labyrinth/environments.
func New(dir string) *Registry {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			dir = "/tmp/labyrinth/environments"
		} else {
			dir = filepath.Join(home, ".labyrinth", "environments")
		}
	}
	return &Registry{dir: dir}
}

// Register writes an environment to the registry directory.
func (r *Registry) Register(env Environment) error {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal environment: %w", err)
	}

	path := filepath.Join(r.dir, env.Name+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	return nil
}

// Load reads an environment by name from the registry.
func (r *Registry) Load(name string) (Environment, error) {
	path := filepath.Join(r.dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Environment{}, fmt.Errorf("environment '%s' not found: %w", name, err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return Environment{}, fmt.Errorf("parse environment '%s': %w", name, err)
	}

	return env, nil
}

// Remove deletes an environment from the registry.
func (r *Registry) Remove(name string) error {
	path := filepath.Join(r.dir, name+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove environment '%s': %w", name, err)
	}
	return nil
}

// ListAll returns all registered environments, sorted by name.
func (r *Registry) ListAll() ([]Environment, error) {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	var envs []Environment
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(r.dir, entry.Name()))
		if err != nil {
			continue
		}

		var env Environment
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		envs = append(envs, env)
	}

	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})

	return envs, nil
}
