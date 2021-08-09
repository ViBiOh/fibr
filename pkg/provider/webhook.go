package provider

// Webhook stores informations about webhook
type Webhook struct {
	Pathname string            `json:"pathname"`
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers,omitempty"`
}
