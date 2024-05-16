package nodemailer

import (
	"bytes"
	"encoding/json"
	"github.com/SourceForgery/tachikoma-bridge/build/generated/github.com/SourceForgery/tachikoma"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"net/url"
)

var Logger zerolog.Logger

type Headers struct {
	XCampaignID string `json:"X-Campaign-Id"`
}

type Message struct {
	Headers map[string]string `json:"headers"`
}

type EventData struct {
	Recipient string  `json:"recipient"`
	Event     string  `json:"event"`
	Message   Message `json:"message"`
}

type Provider struct {
	EventData EventData `json:"event-data"`
}

func SendEvent(event *tachikoma.EmailNotification, parselvoyUri *url.URL) {
	//url := "http://localhost:3000/api/providers/J7O98WX4ZL/json"

	eventType := "something undefined"
	if clicked := event.GetClickedEvent(); clicked != nil {
		eventType = "clicked"
	}

	data := Provider{
		EventData: EventData{
			Recipient: event.RecipientEmailAddress.Email,
			Event:     eventType,
			Message: Message{
				Headers: event.GetEmailTrackingData().Metadata,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		Logger.Fatal().Err(err).Msg("Error occurred during marshalling: ")
	}

	req, err := http.NewRequest("POST", parselvoyUri.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		Logger.Fatal().Err(err).Msg("Error on creating request object: ")
	}

	Logger.Debug().Any("msg", data).Msg(string(jsonData))

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		Logger.Fatal().Err(err).Msg("Error on dispatching request: ")
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
}
