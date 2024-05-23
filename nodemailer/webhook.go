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
	// 'clicked' | 'opened' | 'bounced' | 'complained' | 'failed'
	eventType := ""
	if clicked := event.GetClickedEvent(); clicked != nil {
		eventType = "clicked"
	} else if opened := event.GetOpenedEvent(); opened != nil {
		eventType = "opened"
	} else if bounced := event.GetHardBouncedEvent(); bounced != nil {
		eventType = "bounced"
	} else if complained := event.GetReportedAbuseEvent(); complained != nil {
		eventType = "complained"
	}

	if eventType == "" {
		return
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger.Error().Err(err).Msg("Failed to read body from response")
	}

	if resp.StatusCode/100 != 2 {
		Logger.Error().Msgf("Failed to notify about, statuscode: %d, body: %s", resp.StatusCode, string(body))
	}

}
