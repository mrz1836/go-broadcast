package sync

// GitHub API rate-limit constants used by the sync preflight gate.
//
// These values are the documented, published GitHub limits — NOT values copied
// from a single screenshot. They were fetched from GitHub's official REST API
// rate-limit documentation and cross-checked against a real `GET /rate_limit`
// response on the retrieval date below. The primary hourly ceiling was confirmed
// live (`resources.core.limit == 5000`). The secondary content-creation limits
// are NOT exposed by `GET /rate_limit` — they only surface as an HTTP 403 +
// Retry-After when tripped — so they are guarded statically against the
// documented caps rather than a live "remaining" value.
//
// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
// Retrieved: 2026-06-07 (cross-checked against `gh api rate_limit` on 2026-06-08)
//
// Re-verify these against the source URL if GitHub revises its published limits.
const (
	// PrimaryHourlyLimit is the primary rate limit for an authenticated user /
	// personal access token / OAuth app: 5,000 requests per hour, shared across
	// all of that identity's requests. Exposed live via
	// `GET /rate_limit` -> resources.core.limit.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07 (live-confirmed: resources.core.limit == 5000 on 2026-06-08)
	PrimaryHourlyLimit = 5000

	// SecondaryContentPerMinute is the documented secondary limit on
	// content-generating (mutating) requests: no more than 80 per minute.
	// Not queryable via `GET /rate_limit`; guarded statically.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	SecondaryContentPerMinute = 80

	// SecondaryContentPerHour is the documented secondary limit on
	// content-generating (mutating) requests: no more than 500 per hour.
	// Not queryable via `GET /rate_limit`; guarded statically.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	SecondaryContentPerHour = 500

	// SecondaryPointsPerMinute is the documented per-endpoint points budget for
	// REST API endpoints: no more than 900 points per minute.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	SecondaryPointsPerMinute = 900

	// PointCostRead is the point cost of a GET/HEAD/OPTIONS request: 1 point.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	PointCostRead = 1

	// PointCostWrite is the point cost of a POST/PATCH/PUT/DELETE request:
	// 5 points.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	PointCostWrite = 5

	// PRCreatePointCost is the point cost of creating a pull request, a
	// POST (content-generating) request: 5 points. This is the dominant
	// per-target cost a sync incurs against the secondary points budget.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	PRCreatePointCost = 5

	// SecondaryCPUSecondsPer60s is the documented CPU-time secondary limit:
	// no more than 90 seconds of CPU time per 60 seconds of real time.
	// Recorded for completeness; the preflight estimator does not model CPU time.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	SecondaryCPUSecondsPer60s = 90

	// MaxConcurrentRequests is the documented concurrency secondary limit:
	// no more than 100 concurrent requests. Recorded for completeness; the
	// preflight estimator does not model concurrency.
	//
	// Source: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
	// Retrieved: 2026-06-07
	MaxConcurrentRequests = 100
)
