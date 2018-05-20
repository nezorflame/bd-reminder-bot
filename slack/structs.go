package slack

// Message describes Slack messages
type Message struct {
	ID           uint64 `json:"id"`
	User         string `json:"user,omitempty"`
	Type         string `json:"type"`
	Conversation string `json:"channel"`
	Text         string `json:"text"`
}

// UserProfile describes Slack user profile
type UserProfile struct {
	ID              string `json:"id,omitempty"`
	Title           string `json:"title"`
	Phone           string `json:"phone"`
	Skype           string `json:"skype"` // used for birthday dates
	RealName        string `json:"real_name"`
	RealNameNorm    string `json:"real_name_norm"`
	DisplayName     string `json:"display_name"`
	DisplayNameNorm string `json:"display_name_norm"`
	Email           string `json:"email"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	ImageOriginal   string `json:"image_original"`
	// skipping all other fields intentionally
}

// Conversation describes Slack conversation
type Conversation struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Created     int64     `json:"created"`
	IsArchived  bool      `json:"is_archived"`
	IsGeneral   bool      `json:"is_general"`
	Creator     string    `json:"creator"`
	NameNorm    string    `json:"name_normalized"`
	IsReadOnly  bool      `json:"is_read_only"`
	IsShared    bool      `json:"is_shared"`
	IsOrgShared bool      `json:"is_org_shared"`
	IsMember    bool      `json:"is_member"`
	IsPrivate   bool      `json:"is_private"`
	Topic       TopicInfo `json:"topic"`
	Purpose     TopicInfo `json:"purpose"`
	// skipping all other fields intentionally
}

// TopicInfo describes topic or purpose
type TopicInfo struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// IM describes Slack instant message channel
type IM struct {
	ID            string `json:"id"`
	Created       int64  `json:"created"`
	IsIM          bool   `json:"is_im"`
	IsOrgShared   bool   `json:"is_org_shared"`
	User          string `json:"user"`
	IsUserDeleted bool   `json:"is_user_deleted"`
	// skipping all other fields intentionally
}
