package provider

type Search struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Query string `json:"query"`
}

func (s Search) IsZero() bool {
	return len(s.Name) != 0
}
