package docker

import (
	"testing"
)

func TestBuildComposeArgs(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		project    string
		subcmd     string
		extra      []string
		wantLen    int
		wantFirst  string
		wantSubcmd string
	}{
		{
			name:       "build",
			file:       "docker-compose.yml",
			project:    "labyrinth-test",
			subcmd:     "build",
			wantLen:    6,
			wantFirst:  "compose",
			wantSubcmd: "build",
		},
		{
			name:       "up with -d",
			file:       "docker-compose.yml",
			project:    "labyrinth-prod",
			subcmd:     "up",
			extra:      []string{"-d"},
			wantLen:    7,
			wantFirst:  "compose",
			wantSubcmd: "up",
		},
		{
			name:       "down with -v",
			file:       "/path/to/compose.yml",
			project:    "labyrinth-staging",
			subcmd:     "down",
			extra:      []string{"-v"},
			wantLen:    7,
			wantFirst:  "compose",
			wantSubcmd: "down",
		},
		{
			name:       "ps with format",
			file:       "docker-compose.yml",
			project:    "labyrinth-test",
			subcmd:     "ps",
			extra:      []string{"--format", "table {{.Name}}"},
			wantLen:    8,
			wantFirst:  "compose",
			wantSubcmd: "ps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildComposeArgs(tt.file, tt.project, tt.subcmd, tt.extra...)

			if len(args) != tt.wantLen {
				t.Errorf("args length = %d, want %d; args = %v", len(args), tt.wantLen, args)
			}

			if args[0] != tt.wantFirst {
				t.Errorf("args[0] = %q, want %q", args[0], tt.wantFirst)
			}

			// Subcommand is at index 5 (compose -f FILE -p PROJECT SUBCMD)
			if args[5] != tt.wantSubcmd {
				t.Errorf("subcmd arg = %q, want %q", args[5], tt.wantSubcmd)
			}
		})
	}
}

func TestComposeProjectName(t *testing.T) {
	comp := NewCompose("docker-compose.yml", "labyrinth-myenv")
	if comp.project != "labyrinth-myenv" {
		t.Errorf("project = %q, want %q", comp.project, "labyrinth-myenv")
	}
	if comp.file != "docker-compose.yml" {
		t.Errorf("file = %q, want %q", comp.file, "docker-compose.yml")
	}
}
