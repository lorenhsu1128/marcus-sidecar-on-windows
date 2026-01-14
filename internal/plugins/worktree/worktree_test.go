package worktree

import (
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		mainWorkdir string
		wantCount   int
		wantNames   []string
		wantBranch  []string
	}{
		{
			name: "single worktree",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-feature
HEAD def456
branch refs/heads/feature
`,
			mainWorkdir: "/home/user/project",
			wantCount:   1,
			wantNames:   []string{"project-feature"},
			wantBranch:  []string{"feature"},
		},
		{
			name: "multiple worktrees",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/feature-a
HEAD def456
branch refs/heads/feature-a

worktree /home/user/feature-b
HEAD ghi789
branch refs/heads/feature-b
`,
			mainWorkdir: "/home/user/project",
			wantCount:   2,
			wantNames:   []string{"feature-a", "feature-b"},
			wantBranch:  []string{"feature-a", "feature-b"},
		},
		{
			name: "detached head",
			output: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/detached
HEAD def456
detached
`,
			mainWorkdir: "/home/user/project",
			wantCount:   1,
			wantNames:   []string{"detached"},
			wantBranch:  []string{"(detached)"},
		},
		{
			name:        "empty output",
			output:      "",
			mainWorkdir: "/home/user/project",
			wantCount:   0,
			wantNames:   nil,
			wantBranch:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees, err := parseWorktreeList(tt.output, tt.mainWorkdir)
			if err != nil {
				t.Fatalf("parseWorktreeList() error = %v", err)
			}

			if len(worktrees) != tt.wantCount {
				t.Errorf("got %d worktrees, want %d", len(worktrees), tt.wantCount)
			}

			for i, wt := range worktrees {
				if i < len(tt.wantNames) && wt.Name != tt.wantNames[i] {
					t.Errorf("worktree[%d].Name = %q, want %q", i, wt.Name, tt.wantNames[i])
				}
				if i < len(tt.wantBranch) && wt.Branch != tt.wantBranch[i] {
					t.Errorf("worktree[%d].Branch = %q, want %q", i, wt.Branch, tt.wantBranch[i])
				}
			}
		})
	}
}
