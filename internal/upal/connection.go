package upal

// ConnectionType identifies the kind of external service a connection targets.
type ConnectionType string

const (
	ConnTypeTelegram ConnectionType = "telegram"
	ConnTypeSlack    ConnectionType = "slack"
	ConnTypeHTTP     ConnectionType = "http"
	ConnTypeSMTP     ConnectionType = "smtp"
)

// Connection stores credentials and configuration for an external service.
type Connection struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     ConnectionType `json:"type"`
	Host     string         `json:"host,omitempty"`
	Port     int            `json:"port,omitempty"`
	Login    string         `json:"login,omitempty"`
	Password string         `json:"password,omitempty"` // encrypted at rest
	Token    string         `json:"token,omitempty"`    // encrypted at rest
	Extras   map[string]any `json:"extras,omitempty"`
}

// ConnectionSafe is the API-safe view of a Connection with secrets masked.
type ConnectionSafe struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   ConnectionType `json:"type"`
	Host   string         `json:"host,omitempty"`
	Port   int            `json:"port,omitempty"`
	Login  string         `json:"login,omitempty"`
	Extras map[string]any `json:"extras,omitempty"`
}

// Safe returns a ConnectionSafe view with secrets removed.
func (c *Connection) Safe() ConnectionSafe {
	return ConnectionSafe{
		ID:     c.ID,
		Name:   c.Name,
		Type:   c.Type,
		Host:   c.Host,
		Port:   c.Port,
		Login:  c.Login,
		Extras: c.Extras,
	}
}
