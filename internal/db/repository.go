package db

import (
	"context"
)

// ClientRepository manages Client CRUD operations
type ClientRepository interface {
	Create(ctx context.Context, client *Client) error
	GetByID(ctx context.Context, id uint) (*Client, error)
	GetByName(ctx context.Context, name string) (*Client, error)
	Update(ctx context.Context, client *Client) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context) ([]*Client, error)
	ListWithOrganizations(ctx context.Context) ([]*Client, error)
}

// OrganizationRepository manages Organization CRUD operations
type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, id uint) (*Organization, error)
	GetByName(ctx context.Context, name string) (*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, clientID uint) ([]*Organization, error)
	ListWithRepos(ctx context.Context, clientID uint) ([]*Organization, error)
	FindOrCreate(ctx context.Context, name string, clientID uint) (*Organization, error)
}

// RepoRepository manages Repo CRUD operations
type RepoRepository interface {
	Create(ctx context.Context, repo *Repo) error
	GetByID(ctx context.Context, id uint) (*Repo, error)
	GetByFullName(ctx context.Context, orgName, repoName string) (*Repo, error)
	Update(ctx context.Context, repo *Repo) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, organizationID uint) ([]*Repo, error)
	FindOrCreateFromFullName(ctx context.Context, fullName string, defaultClientID uint) (*Repo, error)
}

// ConfigRepository manages Config CRUD operations
type ConfigRepository interface {
	Create(ctx context.Context, config *Config) error
	GetByID(ctx context.Context, id uint) (*Config, error)
	GetByExternalID(ctx context.Context, externalID string) (*Config, error)
	Update(ctx context.Context, config *Config) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]*Config, error)
}

// GroupRepository manages Group CRUD operations with full preloading
type GroupRepository interface {
	Create(ctx context.Context, group *Group) error
	GetByID(ctx context.Context, id uint) (*Group, error)
	GetByExternalID(ctx context.Context, externalID string) (*Group, error)
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, configID uint) ([]*Group, error)
	// ListWithAssociations preloads Source, Global, Defaults, Targets, Dependencies
	ListWithAssociations(ctx context.Context, configID uint) ([]*Group, error)
}

// TargetRepository manages Target CRUD operations with ref management
type TargetRepository interface {
	Create(ctx context.Context, target *Target) error
	GetByID(ctx context.Context, id uint) (*Target, error)
	GetByRepoName(ctx context.Context, groupID uint, repoFullName string) (*Target, error)
	Update(ctx context.Context, target *Target) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, groupID uint) ([]*Target, error)
	// ListWithAssociations preloads Files, Directories, Transform, FileListRefs, DirectoryListRefs
	ListWithAssociations(ctx context.Context, groupID uint) ([]*Target, error)
	// AddFileListRef adds a file list reference to a target
	AddFileListRef(ctx context.Context, targetID, fileListID uint, position int) error
	// RemoveFileListRef removes a file list reference from a target
	RemoveFileListRef(ctx context.Context, targetID, fileListID uint) error
	// AddDirectoryListRef adds a directory list reference to a target
	AddDirectoryListRef(ctx context.Context, targetID, directoryListID uint, position int) error
	// RemoveDirectoryListRef removes a directory list reference from a target
	RemoveDirectoryListRef(ctx context.Context, targetID, directoryListID uint) error
}

// FileListRepository manages FileList CRUD operations
type FileListRepository interface {
	Create(ctx context.Context, fileList *FileList) error
	GetByID(ctx context.Context, id uint) (*FileList, error)
	GetByExternalID(ctx context.Context, externalID string) (*FileList, error)
	Update(ctx context.Context, fileList *FileList) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, configID uint) ([]*FileList, error)
	// ListWithFiles preloads all FileMappings
	ListWithFiles(ctx context.Context, configID uint) ([]*FileList, error)
}

// DirectoryListRepository manages DirectoryList CRUD operations
type DirectoryListRepository interface {
	Create(ctx context.Context, directoryList *DirectoryList) error
	GetByID(ctx context.Context, id uint) (*DirectoryList, error)
	GetByExternalID(ctx context.Context, externalID string) (*DirectoryList, error)
	Update(ctx context.Context, directoryList *DirectoryList) error
	Delete(ctx context.Context, id uint, hard bool) error
	List(ctx context.Context, configID uint) ([]*DirectoryList, error)
	// ListWithDirectories preloads all DirectoryMappings
	ListWithDirectories(ctx context.Context, configID uint) ([]*DirectoryList, error)
}

// QueryRepository provides cross-entity query operations
type QueryRepository interface {
	// FindByFile finds all targets that sync a specific file path
	FindByFile(ctx context.Context, filePath string) ([]*Target, error)
	// FindByRepo finds all file/directory mappings for a specific repo
	FindByRepo(ctx context.Context, repo string) (*Target, error)
	// FindByFileList finds all targets that reference a specific file list
	FindByFileList(ctx context.Context, fileListID uint) ([]*Target, error)
	// FindByDirectoryList finds all targets that reference a specific directory list
	FindByDirectoryList(ctx context.Context, directoryListID uint) ([]*Target, error)
	// FindByPattern searches file paths matching a pattern
	FindByPattern(ctx context.Context, pattern string) ([]*FileMapping, error)
}
