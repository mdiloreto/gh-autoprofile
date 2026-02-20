package ghauth

import (
	"fmt"
	"os/exec"
	"strings"
)

// UserInfo holds information about a logged-in GitHub account.
type UserInfo struct {
	User     string
	Host     string
	Active   bool
	Protocol string
}

// GetToken retrieves the OAuth token for a specific gh user from the keyring
// without changing the active account.
func GetToken(user string) (string, error) {
	cmd := exec.Command("gh", "auth", "token", "--user", user)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cannot get token for user '%s': %w (is the user logged in via 'gh auth login'?)", user, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ValidateUser checks that a gh user is authenticated and a token can be retrieved.
func ValidateUser(user string) error {
	token, err := GetToken(user)
	if err != nil {
		return fmt.Errorf("user '%s' is not authenticated with gh CLI: %w\nRun: gh auth login", user, err)
	}
	if token == "" {
		return fmt.Errorf("user '%s' returned an empty token — re-authenticate with: gh auth login", user)
	}
	return nil
}

// ListUsers parses `gh auth status` output to list all logged-in accounts.
func ListUsers() ([]UserInfo, error) {
	cmd := exec.Command("gh", "auth", "status")
	// gh auth status exits non-zero when there are inactive accounts,
	// but still prints all info — so we always parse the output.
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil && !strings.Contains(output, "Logged in") {
		return nil, fmt.Errorf("cannot get auth status: %w\nOutput: %s", err, output)
	}
	return parseAuthStatus(output), nil
}

// GetGHVersion returns the gh CLI version string (e.g., "2.86.0").
func GetGHVersion() (string, error) {
	cmd := exec.Command("gh", "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh CLI not found: %w", err)
	}
	// Output: "gh version 2.86.0 (2025-02-18)\n..."
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("unexpected gh --version output")
	}
	parts := strings.Fields(lines[0])
	if len(parts) >= 3 {
		return parts[2], nil
	}
	return lines[0], nil
}

// parseAuthStatus extracts user info from `gh auth status` output.
func parseAuthStatus(output string) []UserInfo {
	var users []UserInfo
	var currentHost string
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Host line: non-indented, contains a dot, no spaces (e.g., "github.com")
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") &&
			strings.Contains(trimmed, ".") && !strings.Contains(trimmed, " ") && trimmed != "" {
			currentHost = trimmed
			continue
		}

		// Account line: "Logged in to github.com account <user> (keyring)"
		if strings.Contains(trimmed, "Logged in to") && strings.Contains(trimmed, "account") {
			parts := strings.SplitN(trimmed, "account ", 2)
			if len(parts) < 2 {
				continue
			}
			rest := parts[1]
			// Username ends at " (" or end of string
			userEnd := strings.Index(rest, " (")
			if userEnd == -1 {
				userEnd = len(rest)
			}
			user := rest[:userEnd]

			// Look ahead for Active and Protocol lines
			active := false
			protocol := "https"
			for j := i + 1; j < len(lines) && j <= i+4; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if strings.Contains(nextLine, "Active account: true") {
					active = true
				}
				if strings.Contains(nextLine, "Active account: false") {
					// explicitly false
				}
				if strings.Contains(nextLine, "Git operations protocol:") {
					colonParts := strings.SplitN(nextLine, ":", 2)
					if len(colonParts) == 2 {
						protocol = strings.TrimSpace(colonParts[1])
					}
				}
				// Stop look-ahead at next account or host
				if strings.Contains(nextLine, "Logged in to") {
					break
				}
			}

			users = append(users, UserInfo{
				User:     user,
				Host:     currentHost,
				Active:   active,
				Protocol: protocol,
			})
		}
	}
	return users
}
