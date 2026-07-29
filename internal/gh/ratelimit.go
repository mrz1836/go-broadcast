package gh

import (
	"context"
	"fmt"
	"time"
)

const (
	// defaultAPIDelay is the default delay between API calls to avoid rate limiting
	defaultAPIDelay = 300 * time.Millisecond
	// maxRetries is the maximum number of retry attempts for rate-limited requests
	maxRetries = 3
)

// initialRetryDelay is the initial backoff delay for retries.
//
// Declared as a variable rather than a constant purely so tests can shorten the
// backoff; production code never reassigns it.
var initialRetryDelay = 2 * time.Second //nolint:gochecknoglobals // tunable so tests need not sleep through real backoff

// rateLimitedDo executes fn with a configurable pre-call delay and exponential backoff retry.
// The delay parameter controls the pre-call wait; use 0 to skip the delay.
// Every error is retried; use rateLimitedDoIf to retry only specific classes of error.
func rateLimitedDo(ctx context.Context, delay time.Duration, fn func() error) error {
	return rateLimitedDoIf(ctx, delay, func(error) bool { return true }, fn)
}

// rateLimitedDoIf behaves like rateLimitedDo but consults shouldRetry before each
// backoff. When shouldRetry reports false the error is returned unwrapped and
// immediately, so callers can still match it with errors.Is/errors.As.
//
// This matters for non-idempotent calls (e.g. creating a pull request) where
// blindly retrying a permanent failure such as HTTP 422 is both useless and slow.
func rateLimitedDoIf(ctx context.Context, delay time.Duration, shouldRetry func(error) bool, fn func() error) error {
	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	var lastErr error
	retryDelay := initialRetryDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Permanent failures surface immediately and unwrapped
		if !shouldRetry(lastErr) {
			return lastErr
		}

		// Don't retry on the last attempt
		if attempt >= maxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
		}
		retryDelay *= 2
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// BuildBranchRuleset constructs a branch protection ruleset from config parameters
func BuildBranchRuleset(name string, include, exclude, rules []string) Ruleset {
	if name == "" {
		name = "branch-protection"
	}

	if include == nil {
		include = []string{}
	}
	if exclude == nil {
		exclude = []string{}
	}

	ruleSpecs := make([]RuleSpec, 0, len(rules))
	for _, r := range rules {
		ruleSpecs = append(ruleSpecs, RuleSpec{Type: r})
	}

	return Ruleset{
		Name:        name,
		Target:      "branch",
		Enforcement: "active",
		BypassActors: []BypassActor{
			{
				ActorID:    5, // Repository admin role
				ActorType:  "RepositoryRole",
				BypassMode: "always",
			},
		},
		Conditions: &RuleConditions{
			RefName: RefNameCondition{
				Include: include,
				Exclude: exclude,
			},
		},
		Rules: ruleSpecs,
	}
}

// BuildTagRuleset constructs a tag protection ruleset from config parameters
func BuildTagRuleset(name string, include, exclude, rules []string) Ruleset {
	if name == "" {
		name = "tag-protection"
	}

	if include == nil {
		include = []string{}
	}
	if exclude == nil {
		exclude = []string{}
	}

	ruleSpecs := make([]RuleSpec, 0, len(rules))
	for _, r := range rules {
		ruleSpecs = append(ruleSpecs, RuleSpec{Type: r})
	}

	return Ruleset{
		Name:        name,
		Target:      "tag",
		Enforcement: "active",
		BypassActors: []BypassActor{
			{
				ActorID:    5,
				ActorType:  "RepositoryRole",
				BypassMode: "always",
			},
		},
		Conditions: &RuleConditions{
			RefName: RefNameCondition{
				Include: include,
				Exclude: exclude,
			},
		},
		Rules: ruleSpecs,
	}
}
