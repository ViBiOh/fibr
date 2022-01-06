package webhook

type slackPayload struct {
	Text   string         `json:"text"`
	Blocks []slackSection `json:"blocks,omitempty"`
}

type slackSection struct {
	Type      string      `json:"type"`
	Text      slackText   `json:"text"`
	Accessory *slackImage `json:"accessory,omitempty"`
	Fields    []slackText `json:"fields,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func newText(value string) slackText {
	return slackText{
		Type: "mrkdwn",
		Text: value,
	}
}

type slackImage struct {
	Type string `json:"type"`
	URL  string `json:"image_url"`
	Alt  string `json:"alt_text"`
}
