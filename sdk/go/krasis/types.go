package krasis

import "time"

// User represents an authenticated user.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Note represents a note document.
type Note struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ContentHTML *string   `json:"content_html,omitempty"`
	OwnerID     string    `json:"owner_id"`
	FolderID    *string   `json:"folder_id,omitempty"`
	Version     int       `json:"version"`
	IsPublic    bool      `json:"is_public"`
	ShareToken  *string   `json:"share_token,omitempty"`
	ViewCount   int       `json:"view_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NoteVersion represents a historical version of a note.
type NoteVersion struct {
	ID            string    `json:"id"`
	NoteID        string    `json:"note_id"`
	Title         *string   `json:"title,omitempty"`
	Content       *string   `json:"content,omitempty"`
	Version       int       `json:"version"`
	ChangedBy     *string   `json:"changed_by,omitempty"`
	ChangeSummary *string   `json:"change_summary,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Folder represents a note folder.
type Folder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	OwnerID   string    `json:"owner_id"`
	Color     *string   `json:"color,omitempty"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ShareStatus represents the status of a shared note.
type ShareStatus struct {
	ShareToken         string     `json:"share_token"`
	ShareURL           string     `json:"share_url"`
	Permission         string     `json:"permission"`
	PasswordProtected  bool       `json:"password_protected"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	Status             string     `json:"status"`
	StatusDescription  string     `json:"status_description"`
	CreatedAt          time.Time `json:"created_at"`
	RejectionReason    *string    `json:"rejection_reason,omitempty"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Highlights []string  `json:"highlights"`
	Score      float64   `json:"score"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Session represents an active login session.
type Session struct {
	SessionID   string    `json:"session_id"`
	DeviceName  string    `json:"device_name"`
	DeviceType  string    `json:"device_type"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	LastActive  time.Time `json:"last_active_at"`
	CreatedAt   time.Time `json:"created_at"`
	IsCurrent   bool      `json:"is_current"`
}

// AskRequest is the request body for AI questions.
type AskRequest struct {
	Question       string `json:"question"`
	ConversationID string `json:"conversation_id,omitempty"`
	ModelID        string `json:"model_id,omitempty"`
	TopK           int    `json:"top_k,omitempty"`
	Stream         bool   `json:"stream,omitempty"`
}

// AIReference is a cited note snippet in an AI response.
type AIReference struct {
	NoteID     string `json:"note_id"`
	NoteTitle  string `json:"note_title"`
	Text       string `json:"text"`
	ChunkIndex int    `json:"chunk_index"`
}

// AskResponse is the response from an AI question.
type AskResponse struct {
	Answer         string         `json:"answer"`
	References     []AIReference  `json:"references,omitempty"`
	ConversationID string         `json:"conversation_id"`
	MessageID      *string        `json:"message_id,omitempty"`
}

// Conversation represents an AI conversation.
type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents a single message in a conversation.
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	References     []any     `json:"references,omitempty"`
	TokenCount     *int      `json:"token_count,omitempty"`
	Model          *string   `json:"model,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// PresignResult is the response from requesting an upload URL.
type PresignResult struct {
	FileID    string `json:"file_id"`
	UploadURL string `json:"upload_url"`
	ExpiresIn int    `json:"expires_in"`
}

// FileItem represents an uploaded file.
type FileItem struct {
	ID          string     `json:"id"`
	NoteID      *string    `json:"note_id,omitempty"`
	UserID      string     `json:"user_id"`
	FileName    string     `json:"file_name"`
	FileType    string     `json:"file_type"`
	StoragePath string     `json:"storage_path"`
	Bucket      string     `json:"bucket"`
	SizeBytes   *int64     `json:"size_bytes,omitempty"`
	Status      int        `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}
