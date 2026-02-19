package gh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockClientComprehensive_ListBranches(t *testing.T) {
	tests := []struct {
		name             string
		repo             string
		expectedBranches []Branch
		expectedError    error
		setupMock        func(*MockClient)
	}{
		{
			name: "successful branch listing",
			repo: "owner/repo",
			expectedBranches: []Branch{
				{
					Name:      "master",
					Protected: true,
					Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{
						SHA: "abc123",
						URL: "https://api.github.com/repos/owner/repo/commits/abc123",
					},
				},
				{
					Name:      "develop",
					Protected: false,
					Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{
						SHA: "def456",
						URL: "https://api.github.com/repos/owner/repo/commits/def456",
					},
				},
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				branches := []Branch{
					{
						Name:      "master",
						Protected: true,
						Commit: struct {
							SHA string `json:"sha"`
							URL string `json:"url"`
						}{
							SHA: "abc123",
							URL: "https://api.github.com/repos/owner/repo/commits/abc123",
						},
					},
					{
						Name:      "develop",
						Protected: false,
						Commit: struct {
							SHA string `json:"sha"`
							URL string `json:"url"`
						}{
							SHA: "def456",
							URL: "https://api.github.com/repos/owner/repo/commits/def456",
						},
					},
				}
				m.On("ListBranches", context.Background(), "owner/repo").Return(branches, nil)
			},
		},
		{
			name:             "empty branch list",
			repo:             "owner/empty-repo",
			expectedBranches: []Branch{},
			expectedError:    nil,
			setupMock: func(m *MockClient) {
				m.On("ListBranches", context.Background(), "owner/empty-repo").Return([]Branch{}, nil)
			},
		},
		{
			name:             "error listing branches",
			repo:             "owner/error-repo",
			expectedBranches: nil,
			expectedError:    assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("ListBranches", context.Background(), "owner/error-repo").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			branches, err := mockClient.ListBranches(context.Background(), tt.repo)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, branches)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBranches, branches)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_GetBranch(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		branch         string
		expectedBranch *Branch
		expectedError  error
		setupMock      func(*MockClient)
	}{
		{
			name:   "successful branch retrieval",
			repo:   "owner/repo",
			branch: "master",
			expectedBranch: &Branch{
				Name:      "master",
				Protected: true,
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{
					SHA: "abc123",
					URL: "https://api.github.com/repos/owner/repo/commits/abc123",
				},
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				branch := &Branch{
					Name:      "master",
					Protected: true,
					Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{
						SHA: "abc123",
						URL: "https://api.github.com/repos/owner/repo/commits/abc123",
					},
				}
				m.On("GetBranch", context.Background(), "owner/repo", "master").Return(branch, nil)
			},
		},
		{
			name:           "branch not found",
			repo:           "owner/repo",
			branch:         "nonexistent",
			expectedBranch: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("GetBranch", context.Background(), "owner/repo", "nonexistent").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			branch, err := mockClient.GetBranch(context.Background(), tt.repo, tt.branch)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, branch)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBranch, branch)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_CreatePR(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		request       PRRequest
		expectedPR    *PR
		expectedError error
		setupMock     func(*MockClient)
	}{
		{
			name: "successful PR creation",
			repo: "owner/repo",
			request: PRRequest{
				Title: "Test PR",
				Body:  "Test PR body",
				Head:  "feature-branch",
				Base:  "master",
			},
			expectedPR: &PR{
				Number: 123,
				State:  "open",
				Title:  "Test PR",
				Body:   "Test PR body",
				Head: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{
					Ref: "feature-branch",
					SHA: "abc123",
				},
				Base: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{
					Ref: "master",
					SHA: "def456",
				},
				User: struct {
					Login string `json:"login"`
				}{
					Login: "testuser",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				request := PRRequest{
					Title: "Test PR",
					Body:  "Test PR body",
					Head:  "feature-branch",
					Base:  "master",
				}
				pr := &PR{
					Number: 123,
					State:  "open",
					Title:  "Test PR",
					Body:   "Test PR body",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "feature-branch",
						SHA: "abc123",
					},
					Base: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "master",
						SHA: "def456",
					},
					User: struct {
						Login string `json:"login"`
					}{
						Login: "testuser",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("CreatePR", context.Background(), "owner/repo", request).Return(pr, nil)
			},
		},
		{
			name: "PR creation error",
			repo: "owner/repo",
			request: PRRequest{
				Title: "Invalid PR",
				Body:  "",
				Head:  "invalid-branch",
				Base:  "master",
			},
			expectedPR:    nil,
			expectedError: assert.AnError,
			setupMock: func(m *MockClient) {
				request := PRRequest{
					Title: "Invalid PR",
					Body:  "",
					Head:  "invalid-branch",
					Base:  "master",
				}
				m.On("CreatePR", context.Background(), "owner/repo", request).Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			pr, err := mockClient.CreatePR(context.Background(), tt.repo, tt.request)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, pr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPR.Number, pr.Number)
				assert.Equal(t, tt.expectedPR.State, pr.State)
				assert.Equal(t, tt.expectedPR.Title, pr.Title)
				assert.Equal(t, tt.expectedPR.Body, pr.Body)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_GetPR(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		number        int
		expectedPR    *PR
		expectedError error
		setupMock     func(*MockClient)
	}{
		{
			name:   "successful PR retrieval",
			repo:   "owner/repo",
			number: 123,
			expectedPR: &PR{
				Number: 123,
				State:  "open",
				Title:  "Test PR",
				Body:   "Test PR body",
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				pr := &PR{
					Number: 123,
					State:  "open",
					Title:  "Test PR",
					Body:   "Test PR body",
				}
				m.On("GetPR", context.Background(), "owner/repo", 123).Return(pr, nil)
			},
		},
		{
			name:          "PR not found",
			repo:          "owner/repo",
			number:        999,
			expectedPR:    nil,
			expectedError: assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("GetPR", context.Background(), "owner/repo", 999).Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			pr, err := mockClient.GetPR(context.Background(), tt.repo, tt.number)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, pr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPR, pr)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_ListPRs(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		state         string
		expectedPRs   []PR
		expectedError error
		setupMock     func(*MockClient)
	}{
		{
			name:  "successful PR listing",
			repo:  "owner/repo",
			state: "open",
			expectedPRs: []PR{
				{
					Number: 123,
					State:  "open",
					Title:  "PR 1",
				},
				{
					Number: 124,
					State:  "open",
					Title:  "PR 2",
				},
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				prs := []PR{
					{
						Number: 123,
						State:  "open",
						Title:  "PR 1",
					},
					{
						Number: 124,
						State:  "open",
						Title:  "PR 2",
					},
				}
				m.On("ListPRs", context.Background(), "owner/repo", "open").Return(prs, nil)
			},
		},
		{
			name:          "empty PR list",
			repo:          "owner/repo",
			state:         "closed",
			expectedPRs:   []PR{},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				m.On("ListPRs", context.Background(), "owner/repo", "closed").Return([]PR{}, nil)
			},
		},
		{
			name:          "error listing PRs",
			repo:          "owner/error-repo",
			state:         "all",
			expectedPRs:   nil,
			expectedError: assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("ListPRs", context.Background(), "owner/error-repo", "all").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			prs, err := mockClient.ListPRs(context.Background(), tt.repo, tt.state)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, prs)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPRs, prs)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_GetFile(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		path          string
		ref           string
		expectedFile  *FileContent
		expectedError error
		setupMock     func(*MockClient)
	}{
		{
			name: "successful file retrieval",
			repo: "owner/repo",
			path: "README.md",
			ref:  "master",
			expectedFile: &FileContent{
				Path:    "README.md",
				Content: []byte("# Test Repository"),
				SHA:     "abc123",
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				file := &FileContent{
					Path:    "README.md",
					Content: []byte("# Test Repository"),
					SHA:     "abc123",
				}
				m.On("GetFile", context.Background(), "owner/repo", "README.md", "master").Return(file, nil)
			},
		},
		{
			name:          "file not found",
			repo:          "owner/repo",
			path:          "nonexistent.txt",
			ref:           "master",
			expectedFile:  nil,
			expectedError: assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("GetFile", context.Background(), "owner/repo", "nonexistent.txt", "master").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			file, err := mockClient.GetFile(context.Background(), tt.repo, tt.path, tt.ref)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFile, file)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClientComprehensive_GetCommit(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		sha            string
		expectedCommit *Commit
		expectedError  error
		setupMock      func(*MockClient)
	}{
		{
			name: "successful commit retrieval",
			repo: "owner/repo",
			sha:  "abc123",
			expectedCommit: &Commit{
				SHA: "abc123",
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
					Message: "Test commit",
					Author: struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					}{
						Name:  "Test User",
						Email: "test@example.com",
						Date:  time.Now(),
					},
				},
			},
			expectedError: nil,
			setupMock: func(m *MockClient) {
				commit := &Commit{
					SHA: "abc123",
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
						Message: "Test commit",
						Author: struct {
							Name  string    `json:"name"`
							Email string    `json:"email"`
							Date  time.Time `json:"date"`
						}{
							Name:  "Test User",
							Email: "test@example.com",
							Date:  time.Now(),
						},
					},
				}
				m.On("GetCommit", context.Background(), "owner/repo", "abc123").Return(commit, nil)
			},
		},
		{
			name:           "commit not found",
			repo:           "owner/repo",
			sha:            "invalid",
			expectedCommit: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockClient) {
				m.On("GetCommit", context.Background(), "owner/repo", "invalid").Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			tt.setupMock(mockClient)

			commit, err := mockClient.GetCommit(context.Background(), tt.repo, tt.sha)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, commit)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCommit.SHA, commit.SHA)
				assert.Equal(t, tt.expectedCommit.Commit.Message, commit.Commit.Message)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestMockClient_ImplementsInterface(t *testing.T) {
	// Test that MockClient implements Client interface
	var _ Client = (*MockClient)(nil)

	// Test instantiation
	mockClient := &MockClient{}
	require.NotNil(t, mockClient)

	// Test that all methods exist and can be called
	ctx := context.Background()

	// Setup mock expectations for interface verification
	mockClient.On("ListBranches", ctx, "test/repo").Return([]Branch{}, nil)
	mockClient.On("GetBranch", ctx, "test/repo", "master").Return(&Branch{}, nil)
	mockClient.On("CreatePR", ctx, "test/repo", PRRequest{}).Return(&PR{}, nil)
	mockClient.On("GetPR", ctx, "test/repo", 1).Return(&PR{}, nil)
	mockClient.On("ListPRs", ctx, "test/repo", "open").Return([]PR{}, nil)
	mockClient.On("GetFile", ctx, "test/repo", "file.txt", "master").Return(&FileContent{}, nil)
	mockClient.On("GetCommit", ctx, "test/repo", "abc123").Return(&Commit{}, nil)

	// Verify all methods work
	_, err := mockClient.ListBranches(ctx, "test/repo")
	require.NoError(t, err)

	_, err = mockClient.GetBranch(ctx, "test/repo", "master")
	require.NoError(t, err)

	_, err = mockClient.CreatePR(ctx, "test/repo", PRRequest{})
	require.NoError(t, err)

	_, err = mockClient.GetPR(ctx, "test/repo", 1)
	require.NoError(t, err)

	_, err = mockClient.ListPRs(ctx, "test/repo", "open")
	require.NoError(t, err)

	_, err = mockClient.GetFile(ctx, "test/repo", "file.txt", "master")
	require.NoError(t, err)

	_, err = mockClient.GetCommit(ctx, "test/repo", "abc123")
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}

// TestNewMockClient tests that NewMockClient returns a non-nil mock
func TestNewMockClient(t *testing.T) {
	t.Parallel()

	m := NewMockClient()
	require.NotNil(t, m)

	// Verify the returned mock satisfies the Client interface
	var _ Client = m
}

// TestMockClient_ClosePR tests the ClosePR mock method
func TestMockClient_ClosePR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ClosePR", mock.Anything, "owner/repo", 42, "closing comment").Return(nil)

		err := m.ClosePR(context.Background(), "owner/repo", 42, "closing comment")
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ClosePR", mock.Anything, "owner/repo", 42, "comment").Return(assert.AnError)

		err := m.ClosePR(context.Background(), "owner/repo", 42, "comment")
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_DeleteBranch tests the DeleteBranch mock method
func TestMockClient_DeleteBranch(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("DeleteBranch", mock.Anything, "owner/repo", "feature-branch").Return(nil)

		err := m.DeleteBranch(context.Background(), "owner/repo", "feature-branch")
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("DeleteBranch", mock.Anything, "owner/repo", "protected").Return(assert.AnError)

		err := m.DeleteBranch(context.Background(), "owner/repo", "protected")
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_UpdatePR tests the UpdatePR mock method
func TestMockClient_UpdatePR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		updates := PRUpdate{Body: ptrString("updated body")}
		m.On("UpdatePR", mock.Anything, "owner/repo", 10, updates).Return(nil)

		err := m.UpdatePR(context.Background(), "owner/repo", 10, updates)
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		updates := PRUpdate{}
		m.On("UpdatePR", mock.Anything, "owner/repo", 10, updates).Return(assert.AnError)

		err := m.UpdatePR(context.Background(), "owner/repo", 10, updates)
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetCurrentUser tests the GetCurrentUser mock method
func TestMockClient_GetCurrentUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := &User{Login: "testuser", ID: 1, Name: "Test User"}
		m.On("GetCurrentUser", mock.Anything).Return(expected, nil)

		user, err := m.GetCurrentUser(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expected, user)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetCurrentUser", mock.Anything).Return(nil, assert.AnError)

		user, err := m.GetCurrentUser(context.Background())
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, user)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetGitTree tests the GetGitTree mock method
func TestMockClient_GetGitTree(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := &GitTree{
			SHA: "abc123",
			Tree: []GitTreeNode{
				{Path: "README.md", Type: "blob", SHA: "def456"},
			},
		}
		m.On("GetGitTree", mock.Anything, "owner/repo", "abc123", true).Return(expected, nil)

		tree, err := m.GetGitTree(context.Background(), "owner/repo", "abc123", true)
		require.NoError(t, err)
		assert.Equal(t, expected, tree)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetGitTree", mock.Anything, "owner/repo", "bad", false).Return(nil, assert.AnError)

		tree, err := m.GetGitTree(context.Background(), "owner/repo", "bad", false)
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, tree)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetRepository tests the GetRepository mock method
func TestMockClient_GetRepository(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := &Repository{
			Name:             "repo",
			FullName:         "owner/repo",
			DefaultBranch:    "main",
			AllowSquashMerge: true,
		}
		m.On("GetRepository", mock.Anything, "owner/repo").Return(expected, nil)

		repo, err := m.GetRepository(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, expected, repo)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetRepository", mock.Anything, "nonexistent/repo").Return(nil, assert.AnError)

		repo, err := m.GetRepository(context.Background(), "nonexistent/repo")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, repo)
		m.AssertExpectations(t)
	})
}

// TestMockClient_ReviewPR tests the ReviewPR mock method
func TestMockClient_ReviewPR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ReviewPR", mock.Anything, "owner/repo", 5, "LGTM").Return(nil)

		err := m.ReviewPR(context.Background(), "owner/repo", 5, "LGTM")
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ReviewPR", mock.Anything, "owner/repo", 5, "").Return(assert.AnError)

		err := m.ReviewPR(context.Background(), "owner/repo", 5, "")
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_MergePR tests the MergePR mock method
func TestMockClient_MergePR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("MergePR", mock.Anything, "owner/repo", 7, MergeMethodSquash).Return(nil)

		err := m.MergePR(context.Background(), "owner/repo", 7, MergeMethodSquash)
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("MergePR", mock.Anything, "owner/repo", 7, MergeMethodMerge).Return(assert.AnError)

		err := m.MergePR(context.Background(), "owner/repo", 7, MergeMethodMerge)
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_BypassMergePR tests the BypassMergePR mock method
func TestMockClient_BypassMergePR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("BypassMergePR", mock.Anything, "owner/repo", 8, MergeMethodRebase).Return(nil)

		err := m.BypassMergePR(context.Background(), "owner/repo", 8, MergeMethodRebase)
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("BypassMergePR", mock.Anything, "owner/repo", 8, MergeMethodSquash).Return(assert.AnError)

		err := m.BypassMergePR(context.Background(), "owner/repo", 8, MergeMethodSquash)
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_EnableAutoMergePR tests the EnableAutoMergePR mock method
func TestMockClient_EnableAutoMergePR(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("EnableAutoMergePR", mock.Anything, "owner/repo", 9, MergeMethodSquash).Return(nil)

		err := m.EnableAutoMergePR(context.Background(), "owner/repo", 9, MergeMethodSquash)
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("EnableAutoMergePR", mock.Anything, "owner/repo", 9, MergeMethodMerge).Return(assert.AnError)

		err := m.EnableAutoMergePR(context.Background(), "owner/repo", 9, MergeMethodMerge)
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_SearchAssignedPRs tests the SearchAssignedPRs mock method
func TestMockClient_SearchAssignedPRs(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []PR{
			{Number: 1, Title: "PR one", State: "open"},
			{Number: 2, Title: "PR two", State: "open"},
		}
		m.On("SearchAssignedPRs", mock.Anything).Return(expected, nil)

		prs, err := m.SearchAssignedPRs(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expected, prs)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("SearchAssignedPRs", mock.Anything).Return(nil, assert.AnError)

		prs, err := m.SearchAssignedPRs(context.Background())
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, prs)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetPRReviews tests the GetPRReviews mock method
func TestMockClient_GetPRReviews(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []Review{
			{ID: 1, State: "APPROVED", User: User{Login: "reviewer1"}},
		}
		m.On("GetPRReviews", mock.Anything, "owner/repo", 3).Return(expected, nil)

		reviews, err := m.GetPRReviews(context.Background(), "owner/repo", 3)
		require.NoError(t, err)
		assert.Equal(t, expected, reviews)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetPRReviews", mock.Anything, "owner/repo", 3).Return(nil, assert.AnError)

		reviews, err := m.GetPRReviews(context.Background(), "owner/repo", 3)
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, reviews)
		m.AssertExpectations(t)
	})
}

// TestMockClient_HasApprovedReview tests the HasApprovedReview mock method
func TestMockClient_HasApprovedReview(t *testing.T) {
	t.Parallel()

	t.Run("approved", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("HasApprovedReview", mock.Anything, "owner/repo", 4, "reviewer").Return(true, nil)

		approved, err := m.HasApprovedReview(context.Background(), "owner/repo", 4, "reviewer")
		require.NoError(t, err)
		assert.True(t, approved)
		m.AssertExpectations(t)
	})

	t.Run("not approved", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("HasApprovedReview", mock.Anything, "owner/repo", 4, "other").Return(false, nil)

		approved, err := m.HasApprovedReview(context.Background(), "owner/repo", 4, "other")
		require.NoError(t, err)
		assert.False(t, approved)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("HasApprovedReview", mock.Anything, "owner/repo", 4, "user").Return(false, assert.AnError)

		approved, err := m.HasApprovedReview(context.Background(), "owner/repo", 4, "user")
		require.ErrorIs(t, err, assert.AnError)
		assert.False(t, approved)
		m.AssertExpectations(t)
	})
}

// TestMockClient_AddPRComment tests the AddPRComment mock method
func TestMockClient_AddPRComment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("AddPRComment", mock.Anything, "owner/repo", 6, "nice work").Return(nil)

		err := m.AddPRComment(context.Background(), "owner/repo", 6, "nice work")
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("AddPRComment", mock.Anything, "owner/repo", 6, "comment").Return(assert.AnError)

		err := m.AddPRComment(context.Background(), "owner/repo", 6, "comment")
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetPRCheckStatus tests the GetPRCheckStatus mock method
func TestMockClient_GetPRCheckStatus(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := &CheckStatusSummary{
			Total:     3,
			Completed: 3,
			Passed:    2,
			Skipped:   1,
		}
		m.On("GetPRCheckStatus", mock.Anything, "owner/repo", 11).Return(expected, nil)

		status, err := m.GetPRCheckStatus(context.Background(), "owner/repo", 11)
		require.NoError(t, err)
		assert.Equal(t, expected, status)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetPRCheckStatus", mock.Anything, "owner/repo", 11).Return(nil, assert.AnError)

		status, err := m.GetPRCheckStatus(context.Background(), "owner/repo", 11)
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, status)
		m.AssertExpectations(t)
	})
}

// TestMockClient_DiscoverOrgRepos tests the DiscoverOrgRepos mock method
func TestMockClient_DiscoverOrgRepos(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []RepoInfo{
			{Name: "repo1", FullName: "org/repo1", DefaultBranch: "main"},
			{Name: "repo2", FullName: "org/repo2", DefaultBranch: "master"},
		}
		m.On("DiscoverOrgRepos", mock.Anything, "org").Return(expected, nil)

		repos, err := m.DiscoverOrgRepos(context.Background(), "org")
		require.NoError(t, err)
		assert.Equal(t, expected, repos)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("DiscoverOrgRepos", mock.Anything, "bad-org").Return(nil, assert.AnError)

		repos, err := m.DiscoverOrgRepos(context.Background(), "bad-org")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, repos)
		m.AssertExpectations(t)
	})
}

// TestMockClient_ExecuteGraphQL tests the ExecuteGraphQL mock method
func TestMockClient_ExecuteGraphQL(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := map[string]interface{}{
			"data": map[string]interface{}{"viewer": map[string]interface{}{"login": "user"}},
		}
		m.On("ExecuteGraphQL", mock.Anything, "{ viewer { login } }").Return(expected, nil)

		result, err := m.ExecuteGraphQL(context.Background(), "{ viewer { login } }")
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		m.AssertExpectations(t)
	})

	t.Run("nil result on error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ExecuteGraphQL", mock.Anything, "bad query").Return(nil, assert.AnError)

		result, err := m.ExecuteGraphQL(context.Background(), "bad query")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetDependabotAlerts tests the GetDependabotAlerts mock method
func TestMockClient_GetDependabotAlerts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []DependabotAlert{
			{Number: 1, State: "open"},
		}
		m.On("GetDependabotAlerts", mock.Anything, "owner/repo").Return(expected, nil)

		alerts, err := m.GetDependabotAlerts(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, expected, alerts)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetDependabotAlerts", mock.Anything, "owner/repo").Return(nil, assert.AnError)

		alerts, err := m.GetDependabotAlerts(context.Background(), "owner/repo")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, alerts)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetCodeScanningAlerts tests the GetCodeScanningAlerts mock method
func TestMockClient_GetCodeScanningAlerts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []CodeScanningAlert{
			{Number: 1, State: "open"},
		}
		m.On("GetCodeScanningAlerts", mock.Anything, "owner/repo").Return(expected, nil)

		alerts, err := m.GetCodeScanningAlerts(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, expected, alerts)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetCodeScanningAlerts", mock.Anything, "owner/repo").Return(nil, assert.AnError)

		alerts, err := m.GetCodeScanningAlerts(context.Background(), "owner/repo")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, alerts)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetSecretScanningAlerts tests the GetSecretScanningAlerts mock method
func TestMockClient_GetSecretScanningAlerts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []SecretScanningAlert{
			{Number: 1, State: "open", SecretType: "github_token"},
		}
		m.On("GetSecretScanningAlerts", mock.Anything, "owner/repo").Return(expected, nil)

		alerts, err := m.GetSecretScanningAlerts(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, expected, alerts)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetSecretScanningAlerts", mock.Anything, "owner/repo").Return(nil, assert.AnError)

		alerts, err := m.GetSecretScanningAlerts(context.Background(), "owner/repo")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, alerts)
		m.AssertExpectations(t)
	})
}

// TestMockClient_ListWorkflows tests the ListWorkflows mock method
func TestMockClient_ListWorkflows(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []Workflow{
			{ID: 1, Name: "CI", Path: ".github/workflows/ci.yml", State: "active"},
		}
		m.On("ListWorkflows", mock.Anything, "owner/repo").Return(expected, nil)

		workflows, err := m.ListWorkflows(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, expected, workflows)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("ListWorkflows", mock.Anything, "owner/repo").Return(nil, assert.AnError)

		workflows, err := m.ListWorkflows(context.Background(), "owner/repo")
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, workflows)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetWorkflowRuns tests the GetWorkflowRuns mock method
func TestMockClient_GetWorkflowRuns(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []WorkflowRun{
			{ID: 100, Name: "CI", Status: "completed", Conclusion: "success"},
		}
		m.On("GetWorkflowRuns", mock.Anything, "owner/repo", int64(1), 5).Return(expected, nil)

		runs, err := m.GetWorkflowRuns(context.Background(), "owner/repo", int64(1), 5)
		require.NoError(t, err)
		assert.Equal(t, expected, runs)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetWorkflowRuns", mock.Anything, "owner/repo", int64(1), 5).Return(nil, assert.AnError)

		runs, err := m.GetWorkflowRuns(context.Background(), "owner/repo", int64(1), 5)
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, runs)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetRunArtifacts tests the GetRunArtifacts mock method
func TestMockClient_GetRunArtifacts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := []Artifact{
			{ID: 200, Name: "coverage", SizeInBytes: 1024},
		}
		m.On("GetRunArtifacts", mock.Anything, "owner/repo", int64(100)).Return(expected, nil)

		artifacts, err := m.GetRunArtifacts(context.Background(), "owner/repo", int64(100))
		require.NoError(t, err)
		assert.Equal(t, expected, artifacts)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetRunArtifacts", mock.Anything, "owner/repo", int64(100)).Return(nil, assert.AnError)

		artifacts, err := m.GetRunArtifacts(context.Background(), "owner/repo", int64(100))
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, artifacts)
		m.AssertExpectations(t)
	})
}

// TestMockClient_DownloadRunArtifact tests the DownloadRunArtifact mock method
func TestMockClient_DownloadRunArtifact(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("DownloadRunArtifact", mock.Anything, "owner/repo", int64(100), "coverage", "/tmp/out").Return(nil)

		err := m.DownloadRunArtifact(context.Background(), "owner/repo", int64(100), "coverage", "/tmp/out")
		require.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("DownloadRunArtifact", mock.Anything, "owner/repo", int64(100), "missing", "/tmp/out").Return(assert.AnError)

		err := m.DownloadRunArtifact(context.Background(), "owner/repo", int64(100), "missing", "/tmp/out")
		require.ErrorIs(t, err, assert.AnError)
		m.AssertExpectations(t)
	})
}

// TestMockClient_GetRateLimit tests the GetRateLimit mock method
func TestMockClient_GetRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		expected := &RateLimitResponse{}
		expected.Resources.Core.Limit = 5000
		expected.Resources.Core.Remaining = 4999
		m.On("GetRateLimit", mock.Anything).Return(expected, nil)

		limit, err := m.GetRateLimit(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 5000, limit.Resources.Core.Limit)
		assert.Equal(t, 4999, limit.Resources.Core.Remaining)
		m.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m := NewMockClient()
		m.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError)

		limit, err := m.GetRateLimit(context.Background())
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, limit)
		m.AssertExpectations(t)
	})
}

// ptrString is a helper to create a string pointer for test data
func ptrString(s string) *string {
	return &s
}
