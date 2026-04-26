package krasis

import (
	"fmt"
	"net/url"
)

// NotesModule handles note CRUD operations.
type NotesModule struct {
	client *Client
}

// NewNotesModule creates a new notes module.
func NewNotesModule(c *Client) *NotesModule {
	return &NotesModule{client: c}
}

// ListNotesOptions for filtering notes.
type ListNotesOptions struct {
	FolderID string
	Page     int
	Size     int
}

// List retrieves a paginated list of notes.
func (n *NotesModule) List(opts *ListNotesOptions) (*Paginated[Note], error) {
	if opts == nil {
		opts = &ListNotesOptions{Page: 1, Size: 20}
	}
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", opts.Page))
	params.Set("size", fmt.Sprintf("%d", opts.Size))
	if opts.FolderID != "" {
		params.Set("folder_id", opts.FolderID)
	}
	var result Paginated[Note]
	err := n.client.Get("/notes?"+params.Encode(), &result)
	return &result, err
}

// Create creates a new note.
func (n *NotesModule) Create(title, content string, folderID *string) (*Note, error) {
	body := map[string]any{"title": title, "content": content}
	if folderID != nil {
		body["folder_id"] = *folderID
	}
	var note Note
	err := n.client.Post("/notes", body, &note)
	return &note, err
}

// Get retrieves a note by ID.
func (n *NotesModule) Get(id string) (*Note, error) {
	var note Note
	err := n.client.Get("/notes/"+id, &note)
	return &note, err
}

// UpdateNoteOptions for updating a note.
type UpdateNoteOptions struct {
	Title         *string
	Content       *string
	FolderID      *string
	IsPublic      *bool
	Version       int
	ChangeSummary *string
}

// Update modifies an existing note.
func (n *NotesModule) Update(id string, opts UpdateNoteOptions) (*Note, error) {
	body := map[string]any{}
	if opts.Title != nil {
		body["title"] = *opts.Title
	}
	if opts.Content != nil {
		body["content"] = *opts.Content
	}
	if opts.FolderID != nil {
		body["folder_id"] = *opts.FolderID
	}
	if opts.IsPublic != nil {
		body["is_public"] = *opts.IsPublic
	}
	if opts.ChangeSummary != nil {
		body["change_summary"] = *opts.ChangeSummary
	}

	headers := map[string]string{}
	if opts.Version > 0 {
		headers["If-Match"] = fmt.Sprintf("%d", opts.Version)
	}

	var note Note
	err := n.client.Put("/notes/"+id, body, &note, headers)
	return &note, err
}

// Delete removes a note.
func (n *NotesModule) Delete(id string) error {
	return n.client.Delete("/notes/"+id, nil)
}

// Versions returns the version history of a note.
func (n *NotesModule) Versions(id string) ([]NoteVersion, error) {
	var versions []NoteVersion
	err := n.client.Get("/notes/"+id+"/versions", &versions)
	return versions, err
}

// RestoreVersion rolls back a note to a previous version.
func (n *NotesModule) RestoreVersion(id string, version int) (*Note, error) {
	var note Note
	err := n.client.Post(fmt.Sprintf("/notes/%s/versions/%d/restore", id, version), nil, &note)
	return &note, err
}

// FoldersModule handles folder CRUD operations.
type FoldersModule struct {
	client *Client
}

// NewFoldersModule creates a new folders module.
func NewFoldersModule(c *Client) *FoldersModule {
	return &FoldersModule{client: c}
}

// List returns all folders.
func (f *FoldersModule) List() ([]Folder, error) {
	var folders []Folder
	err := f.client.Get("/folders", &folders)
	return folders, err
}

// Create creates a new folder.
func (f *FoldersModule) Create(name string, parentID, color *string, sortOrder int) (*Folder, error) {
	body := map[string]any{"name": name, "sort_order": sortOrder}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	if color != nil {
		body["color"] = *color
	}
	var folder Folder
	err := f.client.Post("/folders", body, &folder)
	return &folder, err
}

// Update modifies an existing folder.
func (f *FoldersModule) Update(id string, name string, parentID, color *string, sortOrder *int) (*Folder, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	if color != nil {
		body["color"] = *color
	}
	if sortOrder != nil {
		body["sort_order"] = *sortOrder
	}
	var folder Folder
	err := f.client.Put("/folders/"+id, body, &folder, nil)
	return &folder, err
}

// Delete removes a folder.
func (f *FoldersModule) Delete(id string) error {
	return f.client.Delete("/folders/"+id, nil)
}

// ShareModule handles note sharing operations.
type ShareModule struct {
	client *Client
}

// NewShareModule creates a new share module.
func NewShareModule(c *Client) *ShareModule {
	return &ShareModule{client: c}
}

// CreateShareOptions for creating a share.
type CreateShareOptions struct {
	Permission string
	Password   string
	ExpiresAt  string
}

// Create generates a share for a note.
func (s *ShareModule) Create(noteID string, opts *CreateShareOptions) (*ShareStatus, error) {
	body := map[string]any{"permission": "read"}
	if opts != nil {
		if opts.Permission != "" {
			body["permission"] = opts.Permission
		}
		if opts.Password != "" {
			body["password"] = opts.Password
		}
		if opts.ExpiresAt != "" {
			body["expires_at"] = opts.ExpiresAt
		}
	}
	var share ShareStatus
	err := s.client.Post("/notes/"+noteID+"/share", body, &share)
	return &share, err
}

// Get returns the share status for a note.
func (s *ShareModule) Get(noteID string) (*ShareStatus, error) {
	var share ShareStatus
	err := s.client.Get("/notes/"+noteID+"/share", &share)
	return &share, err
}

// Revoke deletes an existing share.
func (s *ShareModule) Revoke(noteID string) error {
	return s.client.Delete("/notes/"+noteID+"/share", nil)
}

// AccessByToken accesses a shared note by token.
func (s *ShareModule) AccessByToken(token string, password string) (*Note, error) {
	path := "/share/" + token
	if password != "" {
		path += "?password=" + url.QueryEscape(password)
	}
	var note Note
	err := s.client.Get(path, &note)
	return &note, err
}
