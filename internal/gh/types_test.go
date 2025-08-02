package gh

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBranch_JSONMarshaling(t *testing.T) {
	branch := Branch{
		Name:      "master",
		Protected: true,
		Commit: struct {
			SHA string `json:"sha"`
			URL string `json:"url"`
		}{
			SHA: "abc123",
			URL: "https://api.github.com/repos/org/repo/commits/abc123",
		},
	}

	// Test marshaling
	data, err := json.Marshal(branch)
	require.NoError(t, err)
	require.Contains(t, string(data), `"name":"master"`)
	require.Contains(t, string(data), `"protected":true`)
	require.Contains(t, string(data), `"sha":"abc123"`)

	// Test unmarshaling
	var decoded Branch
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, branch.Name, decoded.Name)
	require.Equal(t, branch.Protected, decoded.Protected)
	require.Equal(t, branch.Commit.SHA, decoded.Commit.SHA)
	require.Equal(t, branch.Commit.URL, decoded.Commit.URL)
}

func TestPR_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mergedAt := now.Add(time.Hour)

	pr := PR{
		Number: 123,
		State:  "open",
		Title:  "Test PR",
		Body:   "This is a test PR",
		Head: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}{
			Ref: "feature-branch",
			SHA: "def456",
		},
		Base: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}{
			Ref: "master",
			SHA: "abc123",
		},
		User: struct {
			Login string `json:"login"`
		}{
			Login: "testuser",
		},
		CreatedAt: now,
		UpdatedAt: now.Add(30 * time.Minute),
		MergedAt:  &mergedAt,
		Labels: []struct {
			Name string `json:"name"`
		}{
			{Name: "bug"},
			{Name: "high-priority"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(pr)
	require.NoError(t, err)

	// Test unmarshaling
	var decoded PR
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, pr.Number, decoded.Number)
	require.Equal(t, pr.State, decoded.State)
	require.Equal(t, pr.Title, decoded.Title)
	require.Equal(t, pr.Body, decoded.Body)
	require.Equal(t, pr.Head.Ref, decoded.Head.Ref)
	require.Equal(t, pr.Head.SHA, decoded.Head.SHA)
	require.Equal(t, pr.Base.Ref, decoded.Base.Ref)
	require.Equal(t, pr.Base.SHA, decoded.Base.SHA)
	require.Equal(t, pr.User.Login, decoded.User.Login)
	require.Equal(t, pr.CreatedAt.Unix(), decoded.CreatedAt.Unix())
	require.Equal(t, pr.UpdatedAt.Unix(), decoded.UpdatedAt.Unix())
	require.NotNil(t, decoded.MergedAt)
	require.Equal(t, pr.MergedAt.Unix(), decoded.MergedAt.Unix())
	require.Len(t, decoded.Labels, 2)
	require.Equal(t, pr.Labels[0].Name, decoded.Labels[0].Name)
	require.Equal(t, pr.Labels[1].Name, decoded.Labels[1].Name)
}

func TestPR_NilMergedAt(t *testing.T) {
	pr := PR{
		Number:    456,
		State:     "open",
		Title:     "Open PR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		MergedAt:  nil,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)
	require.Contains(t, string(data), `"merged_at":null`)

	var decoded PR
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Nil(t, decoded.MergedAt)
}

func TestPRRequest_JSONMarshaling(t *testing.T) {
	req := PRRequest{
		Title:         "New Feature",
		Body:          "This PR adds a new feature",
		Head:          "feature-branch",
		Base:          "master",
		Labels:        []string{"enhancement", "bug-fix"},
		Assignees:     []string{"user1", "user2"},
		Reviewers:     []string{"reviewer1"},
		TeamReviewers: []string{"team1"},
	}

	// Test marshaling
	data, err := json.Marshal(req)
	require.NoError(t, err)
	require.Contains(t, string(data), `"title":"New Feature"`)
	require.Contains(t, string(data), `"body":"This PR adds a new feature"`)
	require.Contains(t, string(data), `"head":"feature-branch"`)
	require.Contains(t, string(data), `"base":"master"`)
	require.Contains(t, string(data), `"labels":["enhancement","bug-fix"]`)
	require.Contains(t, string(data), `"assignees":["user1","user2"]`)

	// Test unmarshaling
	var decoded PRRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, req.Title, decoded.Title)
	require.Equal(t, req.Body, decoded.Body)
	require.Equal(t, req.Head, decoded.Head)
	require.Equal(t, req.Base, decoded.Base)
	require.Equal(t, req.Labels, decoded.Labels)
	require.Equal(t, req.Assignees, decoded.Assignees)
	require.Equal(t, req.Reviewers, decoded.Reviewers)
	require.Equal(t, req.TeamReviewers, decoded.TeamReviewers)
}

// TestPRRequest_LabelsHandling tests various scenarios for PR labels
func TestPRRequest_LabelsHandling(t *testing.T) {
	t.Run("empty labels are omitted from JSON", func(t *testing.T) {
		req := PRRequest{
			Title: "Test PR",
			Body:  "Test body",
			Head:  "test-branch",
			Base:  "master",
			// Labels is nil/empty
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		// Should not contain labels field due to omitempty
		require.NotContains(t, string(data), `"labels"`)
	})

	t.Run("single label works correctly", func(t *testing.T) {
		req := PRRequest{
			Title:  "Test PR",
			Body:   "Test body",
			Head:   "test-branch",
			Base:   "master",
			Labels: []string{"single-label"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		require.Contains(t, string(data), `"labels":["single-label"]`)

		var decoded PRRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, []string{"single-label"}, decoded.Labels)
	})

	t.Run("multiple labels work correctly", func(t *testing.T) {
		req := PRRequest{
			Title:  "Test PR",
			Body:   "Test body",
			Head:   "test-branch",
			Base:   "master",
			Labels: []string{"label1", "label2", "label3"},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		require.Contains(t, string(data), `"labels":["label1","label2","label3"]`)

		var decoded PRRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, []string{"label1", "label2", "label3"}, decoded.Labels)
	})

	t.Run("empty slice is omitted due to omitempty", func(t *testing.T) {
		req := PRRequest{
			Title:  "Test PR",
			Body:   "Test body",
			Head:   "test-branch",
			Base:   "master",
			Labels: []string{}, // Explicitly empty slice - will be omitted due to omitempty
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		// Empty slice with omitempty is omitted from JSON
		require.NotContains(t, string(data), `"labels"`)

		var decoded PRRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		// After unmarshaling without labels field, slice will be nil
		require.Nil(t, decoded.Labels)
	})
}

func TestCommit_JSONMarshaling(t *testing.T) {
	commitTime := time.Now().UTC().Truncate(time.Second)

	commit := Commit{
		SHA: "abc123def456",
		Commit: struct {
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
		}{
			Message: "Add new feature",
			Author: struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			}{
				Name:  "John Doe",
				Email: "john@example.com",
				Date:  commitTime,
			},
			Committer: struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			}{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Date:  commitTime.Add(time.Minute),
			},
		},
		Parents: []struct {
			SHA string `json:"sha"`
		}{
			{SHA: "parent1"},
			{SHA: "parent2"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(commit)
	require.NoError(t, err)

	// Test unmarshaling
	var decoded Commit
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, commit.SHA, decoded.SHA)
	require.Equal(t, commit.Commit.Message, decoded.Commit.Message)
	require.Equal(t, commit.Commit.Author.Name, decoded.Commit.Author.Name)
	require.Equal(t, commit.Commit.Author.Email, decoded.Commit.Author.Email)
	require.Equal(t, commit.Commit.Author.Date.Unix(), decoded.Commit.Author.Date.Unix())
	require.Equal(t, commit.Commit.Committer.Name, decoded.Commit.Committer.Name)
	require.Equal(t, commit.Commit.Committer.Email, decoded.Commit.Committer.Email)
	require.Equal(t, commit.Commit.Committer.Date.Unix(), decoded.Commit.Committer.Date.Unix())
	require.Len(t, decoded.Parents, 2)
	require.Equal(t, commit.Parents[0].SHA, decoded.Parents[0].SHA)
	require.Equal(t, commit.Parents[1].SHA, decoded.Parents[1].SHA)
}

func TestFile_JSONMarshaling(t *testing.T) {
	file := File{
		Path:     "src/main.go",
		Mode:     "100644",
		Type:     "blob",
		SHA:      "abc123",
		Size:     1234,
		URL:      "https://api.github.com/repos/org/repo/git/blobs/abc123",
		Content:  "cGFja2FnZSBtYWluCg==",
		Encoding: "base64",
	}

	// Test marshaling
	data, err := json.Marshal(file)
	require.NoError(t, err)
	require.Contains(t, string(data), `"path":"src/main.go"`)
	require.Contains(t, string(data), `"mode":"100644"`)
	require.Contains(t, string(data), `"type":"blob"`)
	require.Contains(t, string(data), `"sha":"abc123"`)
	require.Contains(t, string(data), `"size":1234`)
	require.Contains(t, string(data), `"encoding":"base64"`)

	// Test unmarshaling
	var decoded File
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, file.Path, decoded.Path)
	require.Equal(t, file.Mode, decoded.Mode)
	require.Equal(t, file.Type, decoded.Type)
	require.Equal(t, file.SHA, decoded.SHA)
	require.Equal(t, file.Size, decoded.Size)
	require.Equal(t, file.URL, decoded.URL)
	require.Equal(t, file.Content, decoded.Content)
	require.Equal(t, file.Encoding, decoded.Encoding)
}

func TestFileContent_JSONMarshaling(t *testing.T) {
	fc := FileContent{
		Path:    "README.md",
		Content: []byte("# Test Project\n\nThis is a test."),
		SHA:     "def456",
	}

	// Test marshaling
	data, err := json.Marshal(fc)
	require.NoError(t, err)
	require.Contains(t, string(data), `"path":"README.md"`)
	require.Contains(t, string(data), `"sha":"def456"`)

	// Test unmarshaling
	var decoded FileContent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, fc.Path, decoded.Path)
	require.Equal(t, fc.Content, decoded.Content)
	require.Equal(t, fc.SHA, decoded.SHA)
}

func TestPR_EmptyLabels(t *testing.T) {
	pr := PR{
		Number: 789,
		State:  "closed",
		Title:  "Closed PR",
		Labels: []struct {
			Name string `json:"name"`
		}{},
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	var decoded PR
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Empty(t, decoded.Labels)
}

func TestCommit_NoParents(t *testing.T) {
	commit := Commit{
		SHA: "initial",
		Parents: []struct {
			SHA string `json:"sha"`
		}{},
	}

	data, err := json.Marshal(commit)
	require.NoError(t, err)

	var decoded Commit
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Empty(t, decoded.Parents)
}

func TestTypes_DefaultValues(t *testing.T) {
	// Test Branch default values
	var branch Branch
	require.Empty(t, branch.Name)
	require.False(t, branch.Protected)
	require.Empty(t, branch.Commit.SHA)
	require.Empty(t, branch.Commit.URL)

	// Test PR default values
	var pr PR
	require.Zero(t, pr.Number)
	require.Empty(t, pr.State)
	require.Empty(t, pr.Title)
	require.Empty(t, pr.Body)
	require.Empty(t, pr.Head.Ref)
	require.Empty(t, pr.Head.SHA)
	require.Empty(t, pr.Base.Ref)
	require.Empty(t, pr.Base.SHA)
	require.Empty(t, pr.User.Login)
	require.True(t, pr.CreatedAt.IsZero())
	require.True(t, pr.UpdatedAt.IsZero())
	require.Nil(t, pr.MergedAt)
	require.Nil(t, pr.Labels)

	// Test PRRequest default values
	var prReq PRRequest
	require.Empty(t, prReq.Title)
	require.Empty(t, prReq.Body)
	require.Empty(t, prReq.Head)
	require.Empty(t, prReq.Base)

	// Test Commit default values
	var commit Commit
	require.Empty(t, commit.SHA)
	require.Empty(t, commit.Commit.Message)
	require.Empty(t, commit.Commit.Author.Name)
	require.Empty(t, commit.Commit.Author.Email)
	require.True(t, commit.Commit.Author.Date.IsZero())
	require.Empty(t, commit.Commit.Committer.Name)
	require.Empty(t, commit.Commit.Committer.Email)
	require.True(t, commit.Commit.Committer.Date.IsZero())
	require.Nil(t, commit.Parents)

	// Test File default values
	var file File
	require.Empty(t, file.Path)
	require.Empty(t, file.Mode)
	require.Empty(t, file.Type)
	require.Empty(t, file.SHA)
	require.Zero(t, file.Size)
	require.Empty(t, file.URL)
	require.Empty(t, file.Content)
	require.Empty(t, file.Encoding)

	// Test FileContent default values
	var fc FileContent
	require.Empty(t, fc.Path)
	require.Nil(t, fc.Content)
	require.Empty(t, fc.SHA)
}
