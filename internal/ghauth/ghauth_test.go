package ghauth

import (
	"testing"
)

func TestParseAuthStatus(t *testing.T) {
	// Simulated gh auth status output
	output := `github.com
  ✓ Logged in to github.com account alice (keyring)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_************************************

  ✓ Logged in to github.com account bob-work (keyring)
  - Active account: false
  - Git operations protocol: https
  - Token: gho_************************************

  ✓ Logged in to github.com account alice-freelance (keyring)
  - Active account: false
  - Git operations protocol: https
  - Token: gho_************************************
`

	users := parseAuthStatus(output)

	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	// First user
	if users[0].User != "alice" {
		t.Errorf("user 0: expected 'alice', got '%s'", users[0].User)
	}
	if !users[0].Active {
		t.Error("user 0: expected active")
	}
	if users[0].Host != "github.com" {
		t.Errorf("user 0: expected host 'github.com', got '%s'", users[0].Host)
	}
	if users[0].Protocol != "https" {
		t.Errorf("user 0: expected protocol 'https', got '%s'", users[0].Protocol)
	}

	// Second user
	if users[1].User != "bob-work" {
		t.Errorf("user 1: expected 'bob-work', got '%s'", users[1].User)
	}
	if users[1].Active {
		t.Error("user 1: expected inactive")
	}

	// Third user
	if users[2].User != "alice-freelance" {
		t.Errorf("user 2: expected 'alice-freelance', got '%s'", users[2].User)
	}
	if users[2].Active {
		t.Error("user 2: expected inactive")
	}
}

func TestParseAuthStatus_Empty(t *testing.T) {
	users := parseAuthStatus("")
	if len(users) != 0 {
		t.Errorf("expected 0 users for empty output, got %d", len(users))
	}
}

func TestParseAuthStatus_SingleUser(t *testing.T) {
	output := `github.com
  ✓ Logged in to github.com account alice (keyring)
  - Active account: true
  - Git operations protocol: ssh
  - Token: gho_****
`
	users := parseAuthStatus(output)
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].User != "alice" {
		t.Errorf("expected 'alice', got '%s'", users[0].User)
	}
	if users[0].Protocol != "ssh" {
		t.Errorf("expected protocol 'ssh', got '%s'", users[0].Protocol)
	}
}
