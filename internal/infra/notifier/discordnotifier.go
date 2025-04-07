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

func (dn *DiscordNotifier) SendNotification(data []model.StampedData) {
	// Ensure we have enough data points to calculate gains.
	if len(data) < 2 {
		fmt.Println("Not enough data to compute session gains.")
		return
	}

	// Compute differences between the first and last stamped data.
	first := data[0]
	last := data[len(data)-1]

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
				xpDiff := s2.XP - s1.XP
				rankDiff := s1.Rank - s2.Rank

				skillNames += s1.Name + "\n"
				xpGains += fmt.Sprintf("+%d\n", xpDiff)
				rankGains += fmt.Sprintf("+%d (#%d)\n", rankDiff, s2.Rank)
				break
			}
		}
	}

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
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Build the overall webhook payload with two embeds.
	webhookData := DiscordWebhook{
		Content:   "Test Grind Session",
		Username:  "Grind Bot",
		AvatarURL: "https://example.com/avatar.png",
		Embeds:    []DiscordEmbed{overviewEmbed},
	}

	// Generate the daylinechart image using the snapshot-chromedp approach.
	dayLineChart, err := generateLargeLineChartImage(data)
	if err != nil {
		fmt.Println("Error Generating Chart:", err)
		// If the image generation fails, send the webhook without the attachment.
		if err := sendDiscordWebhook(dn.webhookUrl, webhookData); err != nil {
			fmt.Println("Error sending webhook:", err)
		} else {
			fmt.Println("Webhook sent successfully!")
		}
	} else {
		// Set a message content and send the webhook with the daylinechart as the attachment.
		content := "Session XP"
		if err := sendDiscordWebhookWithAttachment(dn.webhookUrl, content, dayLineChart, "my-chart.png", webhookData); err != nil {
			fmt.Println("Error sending webhook with attachment:", err)
		} else {
			fmt.Println("Webhook sent successfully!")
		}
	}

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
func sendDiscordWebhookWithAttachment(webhookURL, content string, fileBytes []byte, fileName string, webhookData DiscordWebhook) error {
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
	start := 0
	if len(history) > 1440 {
		start = len(history) - 1440
	}
	limitedHistory := history[start:]

	// Create a new line chart.
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:           "1920px", // Responsive width
			Height:          "1080px", // Increased height for clarity
			BackgroundColor: "#000000",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
		}),
	)

	// Prepare the X-axis (timestamps in HH:MM format).
	var timestamps []string
	for _, item := range limitedHistory {
		timestamps = append(timestamps, item.Timestamp.Format("15:04"))
	}
	line.SetXAxis(timestamps)

	// Add one series for each skill (skipping "Overall").
	for _, skill := range history[0].Skills {
		if skill.Name == "Overall" {
			continue
		}
		var rankData []opts.LineData
		for _, item := range limitedHistory {
			// Find the matching skill rank in this record.
			var rank int32
			for _, s := range item.Skills {
				if s.Name == skill.Name {
					rank = s.XP
					break
				}
			}
			rankData = append(rankData, opts.LineData{Value: rank})
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
