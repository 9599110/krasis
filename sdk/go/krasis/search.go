package krasis

import (
	"fmt"
	"net/url"
)

// SearchModule handles full-text search and file operations.
type SearchModule struct {
	client *Client
}

// NewSearchModule creates a new search module.
func NewSearchModule(c *Client) *SearchModule {
	return &SearchModule{client: c}
}

// SearchOptions for querying.
type SearchOptions struct {
	Page int
	Size int
	Type string
}

// Query performs a full-text search.
func (s *SearchModule) Query(q string, opts *SearchOptions) ([]SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{Page: 1, Size: 20}
	}
	params := url.Values{}
	params.Set("q", q)
	params.Set("page", fmt.Sprintf("%d", opts.Page))
	params.Set("size", fmt.Sprintf("%d", opts.Size))
	if opts.Type != "" {
		params.Set("type", opts.Type)
	}
	var wrapper struct {
		Items []SearchResult `json:"items"`
		Total int            `json:"total"`
		Page  int            `json:"page"`
		Size  int            `json:"size"`
	}
	err := s.client.Get("/search?"+params.Encode(), &wrapper)
	return wrapper.Items, err
}

// FileModule handles file upload operations.
type FileModule struct {
	client *Client
}

// NewFileModule creates a new file module.
func NewFileModule(c *Client) *FileModule {
	return &FileModule{client: c}
}

// PresignUpload requests a presigned upload URL.
func (f *FileModule) PresignUpload(fileName, fileType string, noteID *string, sizeBytes *int64) (*PresignResult, error) {
	body := map[string]any{
		"file_name": fileName,
		"file_type": fileType,
	}
	if noteID != nil {
		body["note_id"] = *noteID
	}
	if sizeBytes != nil {
		body["size_bytes"] = *sizeBytes
	}
	var result PresignResult
	err := f.client.Post("/files/presign", body, &result)
	return &result, err
}

// ConfirmUpload confirms a completed upload.
func (f *FileModule) ConfirmUpload(fileID string) error {
	return f.client.Post("/files/"+fileID+"/confirm", nil, nil)
}

// Delete removes an uploaded file.
func (f *FileModule) Delete(fileID string) error {
	return f.client.Delete("/files/"+fileID, nil)
}
