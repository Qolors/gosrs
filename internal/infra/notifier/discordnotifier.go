package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/qolors/gosrs/internal/core/model"
)

type DiscordNotifier struct {
	webhookUrl string
}

func NewDiscordNotifier(whUrl string) *DiscordNotifier {
	return &DiscordNotifier{webhookUrl: whUrl}
}

func (dn *DiscordNotifier) SendNotification(day_data []model.StampedData, session_data []model.StampedData) error {
	// Ensure we have enough data points to calculate gains.
	var first model.StampedData
	var last model.StampedData

	if len(session_data) < 2 {

		first = session_data[0]
		//-1 for ending session, -1 for packed data, -1 for zero index = -3
		last = day_data[len(day_data)-3]
	} else {
		first = session_data[0]
		last = session_data[len(session_data)-1]
	}

	// Compute differences between the first and last stamped data.

	var skillNames, xpGains, rankGains string
	for _, s1 := range first.Skills {
		// Skip unwanted skills.
		if s1.Name == "Overall" {
			continue
		}
		// Find the matching skill in the last record.
		for _, s2 := range last.Skills {
			if s2.Name == s1.Name {

				if s1.XP-s2.XP == 0 {
					continue
				}
				xpDiff := s1.XP - s2.XP
				rankDiff := s2.Rank - s1.Rank

				skillNames += s1.Name + "\n"
				xpGains += fmt.Sprintf("+%d\n", xpDiff)
				rankGains += fmt.Sprintf("+%d (#%d)\n", rankDiff, s2.Rank)
				break
			}
		}
	}

	top10 := day_data[len(day_data)-10:]
	// Generate the daylinechart image using the snapshot-chromedp approach.
	bytes, err := generateLargeLineChartImage(top10)

	image := DiscordEmbedImage{URL: "attachment://my-chart.png"}

	// Build the first embed ("XP Overview") with the session details.
	overviewEmbed := DiscordEmbed{
		Title: "XP Overview",
		Color: 14500675,
		Thumbnail: &DiscordEmbedThumbnail{
			URL: "https://i.imgur.com/1NqptGr.png",
		},
		Fields: []DiscordEmbedField{
			{
				Name:   "Skill",
				Value:  skillNames,
				Inline: true,
			},
			{
				Name:   "XP Gain",
				Value:  xpGains,
				Inline: true,
			},
			{
				Name:   "Rank Gain",
				Value:  rankGains,
				Inline: true,
			},
		},
		Image:     &image,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Build the overall webhook payload with two embeds.
	webhookData := DiscordWebhook{
		Content:   "Test Grind Session",
		Username:  "Grind Bot",
		AvatarURL: "https://example.com/avatar.png",
		Embeds:    []DiscordEmbed{overviewEmbed},
	}

	if err := sendDiscordWebhookWithAttachment(dn.webhookUrl, bytes, "my-chart.png", webhookData); err != nil {
		fmt.Print("Webhook Issue")

	}

	return err

}

// DiscordWebhook is the overall payload.
type DiscordWebhook struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents an embed in the payload.
type DiscordEmbed struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Color       int                    `json:"color,omitempty"`
	Thumbnail   *DiscordEmbedThumbnail `json:"thumbnail,omitempty"`
	Fields      []DiscordEmbedField    `json:"fields,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
	Image       *DiscordEmbedImage     `json:"image,omitempty"`
}

// DiscordEmbedThumbnail holds the thumbnail URL.
type DiscordEmbedThumbnail struct {
	URL string `json:"url,omitempty"`
}

// DiscordEmbedField is used for each field in an embed.
type DiscordEmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedImage holds an image URL.
type DiscordEmbedImage struct {
	URL string `json:"url,omitempty"`
}

// sendDiscordWebhook posts the webhook data to the specified Discord webhook URL.
func sendDiscordWebhook(webhookURL string, webhookData DiscordWebhook) error {
	// Marshal the webhook data to JSON.
	payloadBytes, err := json.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook data: %v", err)
	}

	// Create a new POST request with the JSON payload.
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-success HTTP status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	return nil
}

// sendDiscordWebhookWithAttachment sends a Discord webhook message with a file attachment.
// The fileBytes parameter is the content of the file (e.g. the HTML chart), and fileName is the attachment's name.
func sendDiscordWebhookWithAttachment(webhookURL string, fileBytes []byte, fileName string, webhookData DiscordWebhook) error {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Marshal the webhook data (including the embed with our image reference).
	payloadBytes, err := json.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create the "payload_json" form field.
	part, err := writer.CreateFormField("payload_json")
	if err != nil {
		return fmt.Errorf("failed to create form field: %w", err)
	}
	if _, err = part.Write(payloadBytes); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	// Create the file attachment form field.
	part, err = writer.CreateFormFile("file", fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = part.Write(fileBytes); err != nil {
		return fmt.Errorf("failed to write file bytes: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", webhookURL, &b)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-success status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func generateLargeLineChartImage(history []model.StampedData) ([]byte, error) {
	if len(history) == 0 {
		return nil, errors.New("no data available")
	}

	// Use only the last 1440 entries (or less if history is shorter).

	// Create a new line chart.
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:           "800px", // Responsive width
			Height:          "400px", // Increased height for clarity
			BackgroundColor: "#000000",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
		}),
	)

	// Prepare the X-axis (timestamps in HH:MM format).
	var timestamps []string
	for _, item := range history {
		timestamps = append(timestamps, item.Timestamp.Format("15:04"))
	}
	line.SetXAxis(timestamps)

	// Add one series for each skill (skipping "Overall").
	for _, skill := range history[0].Skills {
		if skill.Name == "Overall" {
			continue
		}
		var rankData []opts.LineData
		for _, item := range history {

			// Find the matching skill rank in this record.
			var rank int64
			var hasnoexp bool
			for _, s := range item.Skills {
				if s.Name == skill.Name {
					if s.XP-skill.XP == 0 {
						hasnoexp = true
						break
					}
					rank = s.XP
					break
				}
			}

			if !hasnoexp {
				rankData = append(rankData, opts.LineData{Value: rank})
			}

		}
		smooth := true
		line.AddSeries(skill.Name, rankData).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: &smooth}))
	}

	// Use the snapshot-chromedp approach to generate the chart image.
	// This function takes the HTML content from line.RenderContent() and outputs a PNG file.
	err := render.MakeChartSnapshot(line.RenderContent(), "my-chart.png")
	if err != nil {
		return nil, fmt.Errorf("failed to make chart snapshot: %w", err)
	}

	// Read the generated PNG file.
	imgBytes, err := os.ReadFile("my-chart.png")
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	return imgBytes, nil
}
