// Package gh provides GitHub API client interfaces and types
package gh

import "time"

// Branch represents a GitHub branch
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Commit    struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

// PR represents a GitHub pull request
type PR struct {
	Number int    `json:"number"`
	State  string `json:"state"` // open, closed
	Title  string `json:"title"`
	Body   string `json:"body"`
	Head   struct {
		Ref string `json:"ref"` // branch name
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"` // target branch
		SHA string `json:"sha"`
	} `json:"base"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// PRRequest represents a request to create a pull request
type PRRequest struct {
	Title         string   `json:"title"`
	Body          string   `json:"body"`
	Head          string   `json:"head"`                     // source branch
	Base          string   `json:"base"`                     // target branch
	Labels        []string `json:"labels,omitempty"`         // Labels to apply to PR
	Assignees     []string `json:"assignees,omitempty"`      // GitHub usernames to assign
	Reviewers     []string `json:"reviewers,omitempty"`      // GitHub usernames to request reviews from
	TeamReviewers []string `json:"team_reviewers,omitempty"` // GitHub team slugs to request reviews from
}

// PRUpdate represents updates to an existing pull request
type PRUpdate struct {
	State *string `json:"state,omitempty"` // "open" or "closed"
	Body  *string `json:"body,omitempty"`  // Updated body content
}

// Commit represents a GitHub commit
type Commit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}

// File represents a file in a GitHub repository
type File struct {
	Path     string `json:"path"`
	Mode     string `json:"mode"`
	Type     string `json:"type"`
	SHA      string `json:"sha"`
	Size     int    `json:"size"`
	URL      string `json:"url"`
	Content  string `json:"content"`  // base64 encoded
	Encoding string `json:"encoding"` // usually "base64"
}

// FileContent represents decoded file content
type FileContent struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
	SHA     string `json:"sha"`
}
