package cli

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnexpectedGHInvocation = errors.New("unexpected gh invocation")

type stubCheckAuthRunner struct {
	responses map[string]stubCheckAuthResponse
	calls     []stubCheckAuthCall
}

type stubCheckAuthResponse struct {
	output []byte
	err    error
}

type stubCheckAuthCall struct {
	args []string
	env  []string
}

func (r *stubCheckAuthRunner) Run(_ context.Context, args, env []string) ([]byte, error) {
	r.calls = append(r.calls, stubCheckAuthCall{
		args: slices.Clone(args),
		env:  slices.Clone(env),
	})

	response, ok := r.responses[strings.Join(args, "\x00")]
	if !ok {
		return nil, errUnexpectedGHInvocation
	}

	return response.output, response.err
}

type secretError struct {
	secret string
}

func (e secretError) Error() string {
	return "remote rejected " + e.secret
}

func TestRunCheckAuthWithRunner_GitHubExitCodesAndScopes(t *testing.T) {
	t.Parallel()

	rejectedToken := strings.Join([]string{"ghp", "rejected", "secret", "value"}, "_")
	successToken := strings.Join([]string{"ghp", "success", "secret", "value"}, "_")
	fineGrainedToken := strings.Join([]string{"ghp", "fine-grained", "secret", "value"}, "_")

	tests := []struct {
		name           string
		env            map[string]string
		responses      map[string]stubCheckAuthResponse
		wantErr        bool
		wantExitCode   int
		wantOutput     []string
		wantNoOutput   []string
		wantCallCount  int
		wantTokenInEnv string
	}{
		{
			name:         "no token exits 1 without calling gh",
			env:          map[string]string{},
			responses:    map[string]stubCheckAuthResponse{},
			wantErr:      true,
			wantExitCode: checkAuthExitNoToken,
			wantOutput: []string{
				"GitHub authentication",
				"token detected: no",
				"login: (unknown)",
				"scopes: (none)",
				"AI provider credentials",
				"anthropic (active): unset, source: (none), hint: (none)",
			},
			wantCallCount: 0,
		},
		{
			name: "token rejected exits 2 and redacts the runner error",
			env: map[string]string{
				"GH_PAT_TOKEN": rejectedToken,
			},
			responses: map[string]stubCheckAuthResponse{
				"api\x00user": {
					err: secretError{secret: rejectedToken},
				},
			},
			wantErr:      true,
			wantExitCode: checkAuthExitTokenRejected,
			wantOutput: []string{
				"token detected: yes",
				"login: (unknown)",
				"scopes: (none)",
			},
			wantNoOutput: []string{
				rejectedToken,
				"remote rejected",
			},
			wantCallCount:  1,
			wantTokenInEnv: "GH_TOKEN=" + rejectedToken,
		},
		{
			name: "authenticated token exits 0 with sorted scopes",
			env: map[string]string{
				"GITHUB_TOKEN": successToken,
			},
			responses: map[string]stubCheckAuthResponse{
				"api\x00user": {
					output: []byte(`{"login":"octocat"}`),
				},
				"api\x00-i\x00user": {
					output: []byte("HTTP/2.0 200 OK\nX-Oauth-Scopes: workflow, repo, read:org\n"),
				},
			},
			wantOutput: []string{
				"token detected: yes",
				"login: octocat",
				"scopes: read:org, repo, workflow",
			},
			wantNoOutput: []string{
				successToken,
			},
			wantCallCount:  2,
			wantTokenInEnv: "GH_TOKEN=" + successToken,
		},
		{
			name: "authenticated token exits 0 when scopes header is absent",
			env: map[string]string{
				"GH_TOKEN": fineGrainedToken,
			},
			responses: map[string]stubCheckAuthResponse{
				"api\x00user": {
					output: []byte(`{"login":"finegrained"}`),
				},
				"api\x00-i\x00user": {
					output: []byte("HTTP/2.0 200 OK\n"),
				},
			},
			wantOutput: []string{
				"token detected: yes",
				"login: finegrained",
				"scopes: (none)",
			},
			wantNoOutput: []string{
				fineGrainedToken,
			},
			wantCallCount:  2,
			wantTokenInEnv: "GH_TOKEN=" + fineGrainedToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			runner := &stubCheckAuthRunner{responses: tt.responses}
			err := runCheckAuthWithRunner(
				context.Background(),
				&out,
				runner,
				getenvFromMap(tt.env),
				[]string{"PATH=/usr/bin", "GH_TOKEN=old-token"},
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantExitCode, ExitCodeForError(err))
			} else {
				require.NoError(t, err)
			}

			output := out.String()
			for _, want := range tt.wantOutput {
				assert.Contains(t, output, want)
			}
			for _, secret := range tt.wantNoOutput {
				assert.NotContains(t, output, secret)
				if err != nil {
					assert.NotContains(t, err.Error(), secret)
				}
			}
			assert.Len(t, runner.calls, tt.wantCallCount)
			if tt.wantTokenInEnv != "" {
				require.NotEmpty(t, runner.calls)
				assert.Contains(t, runner.calls[0].env, tt.wantTokenInEnv)
				assert.NotContains(t, runner.calls[0].env, "GH_TOKEN=old-token")
			}
		})
	}
}

func TestRunCheckAuthWithRunner_ParseFailuresAreRejectedAndRedacted(t *testing.T) {
	t.Parallel()

	secret := strings.Join([]string{"ghp", "parse-failure", "secret", "value"}, "_")
	var out bytes.Buffer
	runner := &stubCheckAuthRunner{
		responses: map[string]stubCheckAuthResponse{
			"api\x00user": {output: []byte(`{"id":123}`)},
		},
	}

	err := runCheckAuthWithRunner(
		context.Background(),
		&out,
		runner,
		getenvFromMap(map[string]string{"GH_PAT_TOKEN": secret}),
		[]string{"PATH=/usr/bin"},
	)

	require.Error(t, err)
	assert.Equal(t, checkAuthExitTokenRejected, ExitCodeForError(err))
	assert.NotContains(t, out.String(), secret)
	assert.NotContains(t, err.Error(), secret)
	assert.Contains(t, out.String(), "token detected: yes")
	assert.Contains(t, out.String(), "login: (unknown)")
}

func TestBuildAIAuthReport(t *testing.T) {
	t.Parallel()

	anthropicSecret := "sk-ant-" + strings.Join([]string{"api03", "secret", "value"}, "-")
	openAISecret := "sk-" + strings.Join([]string{"openai", "secret", "value"}, "-")
	googleSecret := "AIza" + strings.Join([]string{"google", "secret", "value"}, "-")
	sharedSecret := strings.Join([]string{"shared", "secret", "value"}, "-")

	tests := []struct {
		name string
		env  map[string]string
		want []aiProviderAuthReport
	}{
		{
			name: "defaults to anthropic active with provider specific keys",
			env: map[string]string{
				"ANTHROPIC_API_KEY": anthropicSecret,
				"OPENAI_API_KEY":    openAISecret,
				"GEMINI_API_KEY":    googleSecret,
			},
			want: []aiProviderAuthReport{
				{
					Provider: "anthropic",
					Active:   true,
					Detected: true,
					Source:   "ANTHROPIC_API_KEY",
					Hint:     "prefix sk-ant-",
				},
				{
					Provider: "openai",
					Detected: true,
					Source:   "OPENAI_API_KEY",
					Hint:     "prefix sk-",
				},
				{
					Provider: "google",
					Detected: true,
					Source:   "GEMINI_API_KEY",
					Hint:     "prefix AIza",
				},
			},
		},
		{
			name: "generic key takes precedence for every provider and active provider is honored",
			env: map[string]string{
				"GO_BROADCAST_AI_PROVIDER": "openai",
				"GO_BROADCAST_AI_API_KEY":  sharedSecret,
				"OPENAI_API_KEY":           openAISecret,
			},
			want: []aiProviderAuthReport{
				{
					Provider: "anthropic",
					Detected: true,
					Source:   "GO_BROADCAST_AI_API_KEY",
					Hint:     "prefix unrecognized",
				},
				{
					Provider: "openai",
					Active:   true,
					Detected: true,
					Source:   "GO_BROADCAST_AI_API_KEY",
					Hint:     "prefix unrecognized",
				},
				{
					Provider: "google",
					Detected: true,
					Source:   "GO_BROADCAST_AI_API_KEY",
					Hint:     "prefix unrecognized",
				},
			},
		},
		{
			name: "unset keys are reported without a source or hint",
			env: map[string]string{
				"GO_BROADCAST_AI_PROVIDER": "google",
			},
			want: []aiProviderAuthReport{
				{
					Provider: "anthropic",
					Hint:     "(none)",
				},
				{
					Provider: "openai",
					Hint:     "(none)",
				},
				{
					Provider: "google",
					Active:   true,
					Hint:     "(none)",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildAIAuthReport(getenvFromMap(tt.env)))
		})
	}
}

func TestPrintAIAuthReportRedactsCredentialValues(t *testing.T) {
	t.Parallel()

	secrets := []string{
		"sk-ant-" + strings.Join([]string{"api03", "secret", "value"}, "-"),
		"sk-" + strings.Join([]string{"openai", "secret", "value"}, "-"),
		"AIza" + strings.Join([]string{"google", "secret", "value"}, "-"),
	}
	var out bytes.Buffer

	printAIAuthReport(&out, buildAIAuthReport(getenvFromMap(map[string]string{
		"ANTHROPIC_API_KEY": secrets[0],
		"OPENAI_API_KEY":    secrets[1],
		"GEMINI_API_KEY":    secrets[2],
	})))

	output := out.String()
	for _, secret := range secrets {
		assert.NotContains(t, output, secret)
	}
	assert.Contains(t, output, "anthropic (active): set, source: ANTHROPIC_API_KEY, hint: prefix sk-ant-")
	assert.Contains(t, output, "openai: set, source: OPENAI_API_KEY, hint: prefix sk-")
	assert.Contains(t, output, "google: set, source: GEMINI_API_KEY, hint: prefix AIza")
}

func getenvFromMap(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
