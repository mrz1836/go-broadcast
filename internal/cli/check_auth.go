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

	"github.com/mrz1836/go-broadcast/internal/ai"
)

const (
	checkAuthExitNoToken       = 1
	checkAuthExitTokenRejected = 2
)

var (
	errCheckAuthNoToken       = errors.New("GitHub token not detected")
	errCheckAuthTokenRejected = errors.New("GitHub authentication check failed")
	errCheckAuthMissingLogin  = errors.New("GitHub user response did not include a login")
)

type checkAuthRunner interface {
	Run(ctx context.Context, args, env []string) ([]byte, error)
}

type ghCheckAuthRunner struct{}

func (r ghCheckAuthRunner) Run(ctx context.Context, args, env []string) ([]byte, error) {
	//nolint:gosec // G204: args are fixed by internal auth probe callers, not user input.
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

type aiProviderAuthReport struct {
	Provider string
	Active   bool
	Detected bool
	Source   string
	Hint     string
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
	aiReport := buildAIAuthReport(getenv)
	if !detected {
		printAuthReport(writer, githubAuthReport{TokenDetected: false}, aiReport)
		return newExitCodeError(checkAuthExitNoToken, errCheckAuthNoToken)
	}

	env := withGitHubTokenEnv(baseEnv, token)
	userOutput, err := runner.Run(ctx, []string{"api", "user"}, env)
	if err != nil {
		printAuthReport(writer, githubAuthReport{
			TokenDetected: true,
			Scopes:        []string{},
		}, aiReport)
		return newExitCodeError(checkAuthExitTokenRejected, errCheckAuthTokenRejected)
	}

	login, err := parseGitHubLogin(userOutput)
	if err != nil {
		printAuthReport(writer, githubAuthReport{
			TokenDetected: true,
			Scopes:        []string{},
		}, aiReport)
		return newExitCodeError(checkAuthExitTokenRejected, errCheckAuthTokenRejected)
	}

	scopesOutput, err := runner.Run(ctx, []string{"api", "-i", "user"}, env)
	scopes := []string{}
	if err == nil {
		scopes = parseGitHubScopes(scopesOutput)
	}

	printAuthReport(writer, githubAuthReport{
		TokenDetected: true,
		Login:         login,
		Scopes:        scopes,
	}, aiReport)

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
		return "", errCheckAuthMissingLogin
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

func buildAIAuthReport(getenv func(string) string) []aiProviderAuthReport {
	activeProvider := strings.TrimSpace(getenv("GO_BROADCAST_AI_PROVIDER"))
	if activeProvider == "" {
		activeProvider = ai.ProviderAnthropic
	}

	reports := make([]aiProviderAuthReport, 0, 3)
	for _, provider := range []string{ai.ProviderAnthropic, ai.ProviderOpenAI, ai.ProviderGoogle} {
		key, source := detectAIProviderKey(getenv, provider)
		reports = append(reports, aiProviderAuthReport{
			Provider: provider,
			Active:   provider == activeProvider,
			Detected: key != "",
			Source:   source,
			Hint:     detectAIKeyHint(provider, key),
		})
	}

	return reports
}

func detectAIProviderKey(getenv func(string) string, provider string) (string, string) {
	if key := strings.TrimSpace(getenv("GO_BROADCAST_AI_API_KEY")); key != "" {
		return key, "GO_BROADCAST_AI_API_KEY"
	}

	source := providerSpecificAIKeyEnv(provider)
	if source == "" {
		return "", ""
	}
	if key := strings.TrimSpace(getenv(source)); key != "" {
		return key, source
	}

	return "", ""
}

func providerSpecificAIKeyEnv(provider string) string {
	switch provider {
	case ai.ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ai.ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ai.ProviderGoogle:
		return "GEMINI_API_KEY"
	default:
		return ""
	}
}

func detectAIKeyHint(provider, key string) string {
	if key == "" {
		return "(none)"
	}

	switch provider {
	case ai.ProviderAnthropic:
		if strings.HasPrefix(key, "sk-ant-") {
			return "prefix sk-ant-"
		}
	case ai.ProviderOpenAI:
		if strings.HasPrefix(key, "sk-") {
			return "prefix sk-"
		}
	case ai.ProviderGoogle:
		if strings.HasPrefix(key, "AIza") {
			return "prefix AIza"
		}
	}

	return "prefix unrecognized"
}

func printAuthReport(writer io.Writer, githubReport githubAuthReport, aiReport []aiProviderAuthReport) {
	printGitHubAuthReport(writer, githubReport)
	_, _ = fmt.Fprintln(writer)
	printAIAuthReport(writer, aiReport)
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

func printAIAuthReport(writer io.Writer, report []aiProviderAuthReport) {
	_, _ = fmt.Fprintf(writer, "AI provider credentials\n")
	for _, provider := range report {
		active := ""
		if provider.Active {
			active = " (active)"
		}

		detected := "unset"
		source := "(none)"
		if provider.Detected {
			detected = "set"
			source = provider.Source
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s%s: %s, source: %s, hint: %s\n",
			provider.Provider,
			active,
			detected,
			source,
			provider.Hint,
		)
	}
}
