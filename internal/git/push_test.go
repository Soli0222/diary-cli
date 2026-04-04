package git

import "testing"

func TestCommitAndPushStagesAndCommitsOnlyTargetFile(t *testing.T) {
	t.Parallel()

	originalRun := run
	defer func() { run = originalRun }()

	var calls [][]string
	run = func(dir string, name string, args ...string) error {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		if dir != "/repo" {
			t.Fatalf("dir = %q, want /repo", dir)
		}
		return nil
	}

	if err := CommitAndPush("/repo", "2026/0404.md", "2026-04-04"); err != nil {
		t.Fatalf("CommitAndPush() error = %v", err)
	}

	want := [][]string{
		{"git", "add", "--", "2026/0404.md"},
		{"git", "commit", "-m", "diary: 2026-04-04", "--only", "--", "2026/0404.md"},
		{"git", "push"},
	}

	if len(calls) != len(want) {
		t.Fatalf("len(calls) = %d, want %d (%v)", len(calls), len(want), calls)
	}
	for i := range want {
		if len(calls[i]) != len(want[i]) {
			t.Fatalf("calls[%d] = %v, want %v", i, calls[i], want[i])
		}
		for j := range want[i] {
			if calls[i][j] != want[i][j] {
				t.Fatalf("calls[%d] = %v, want %v", i, calls[i], want[i])
			}
		}
	}
}
