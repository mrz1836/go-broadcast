// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const (
	checkAuthExitNoToken       = 1
	checkAuthExitTokenRejected = 2
)

var (
	errCheckAuthNoToken       = errors.New("GitHub token not detected")
	errCheckAuthTokenRejected = errors.New("GitHub authentication check failed")
)

type checkAuthRunner interface {
	Run(ctx context.Context, args, env []string) ([]byte, error)
}

type ghCheckAuthRunner struct{}

func (r ghCheckAuthRunner) Run(ctx context.Context, args, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Env = env
	return cmd.Output()
}

type githubUserResponse struct {
	Login string `json:"login"`
}

type githubAuthReport struct {
	TokenDetected bool
	Login         string
	Scopes        []string
}

func runCheckAuth(ctx context.Context, writer io.Writer) error {
	return runCheckAuthWithRunner(ctx, writer, ghCheckAuthRunner{}, os.Getenv, os.Environ())
}

func runCheckAuthWithRunner(
	ctx context.Context,
	writer io.Writer,
	runner checkAuthRunner,
	getenv func(string) string,
	baseEnv []string,
) error {
	token, detected := detectGitHubToken(getenv)
	if !detected {
		printGitHubAuthReport(writer, githubAuthReport{TokenDetected: false})
		return newExitCodeError(checkAuthExitNoToken, errCheckAuthNoToken)
	}

	env := withGitHubTokenEnv(baseEnv, token)
	userOutput, err := runner.Run(ctx, []string{"api", "user"}, env)
	if err != nil {
		printGitHubAuthReport(writer, githubAuthReport{
			TokenDetected: true,
			Scopes:        []string{},
		})
		return newExitCodeError(checkAuthExitTokenRejected, errCheckAuthTokenRejected)
	}

	login, err := parseGitHubLogin(userOutput)
	if err != nil {
		printGitHubAuthReport(writer, githubAuthReport{
			TokenDetected: true,
			Scopes:        []string{},
		})
		return newExitCodeError(checkAuthExitTokenRejected, errCheckAuthTokenRejected)
	}

	scopesOutput, err := runner.Run(ctx, []string{"api", "-i", "user"}, env)
	scopes := []string{}
	if err == nil {
		scopes = parseGitHubScopes(scopesOutput)
	}

	printGitHubAuthReport(writer, githubAuthReport{
		TokenDetected: true,
		Login:         login,
		Scopes:        scopes,
	})

	return nil
}

func detectGitHubToken(getenv func(string) string) (string, bool) {
	for _, key := range []string{"GH_PAT_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if token := strings.TrimSpace(getenv(key)); token != "" {
			return token, true
		}
	}
	return "", false
}

func withGitHubTokenEnv(baseEnv []string, token string) []string {
	env := make([]string, 0, len(baseEnv)+1)
	for _, entry := range baseEnv {
		if strings.HasPrefix(entry, "GH_TOKEN=") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "GH_TOKEN="+token)
	return env
}

func parseGitHubLogin(data []byte) (string, error) {
	var user githubUserResponse
	if err := json.Unmarshal(data, &user); err != nil {
		return "", fmt.Errorf("parse GitHub user response: %w", err)
	}
	if strings.TrimSpace(user.Login) == "" {
		return "", errors.New("GitHub user response did not include a login")
	}
	return user.Login, nil
}

func parseGitHubScopes(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "X-Oauth-Scopes") {
			continue
		}
		scopes := splitScopes(value)
		sort.Strings(scopes)
		return scopes
	}
	return []string{}
}

func splitScopes(value string) []string {
	parts := strings.Split(value, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

func printGitHubAuthReport(writer io.Writer, report githubAuthReport) {
	tokenDetected := "no"
	if report.TokenDetected {
		tokenDetected = "yes"
	}

	login := "(unknown)"
	if strings.TrimSpace(report.Login) != "" {
		login = report.Login
	}

	scopes := "(none)"
	if len(report.Scopes) > 0 {
		scopes = strings.Join(report.Scopes, ", ")
	}

	_, _ = fmt.Fprintf(writer, "GitHub authentication\n")
	_, _ = fmt.Fprintf(writer, "token detected: %s\n", tokenDetected)
	_, _ = fmt.Fprintf(writer, "login: %s\n", login)
	_, _ = fmt.Fprintf(writer, "scopes: %s\n", scopes)
}
