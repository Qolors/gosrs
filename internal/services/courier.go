package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/qolors/gosrs/internal/osrsclient"
)

type Courier struct {
	t      *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc
}

func NewCourier() *Courier {
	ctx, cancel := context.WithCancel(context.Background())
	return &Courier{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Courier) Hitch(timer *time.Ticker) {
	c.t = timer
}

func (c *Courier) Start() {
	go func() {
		for {
			select {
			case <-c.t.C:
				history, err := PullAll()
				if err != nil {
					log.Println("Error polling stats:", err)
					continue
				}
				if len(history.Items) == 0 {
					log.Println("No history items found")
					continue
				}
				// Generate a separate HTML page (line chart) for each Skill and Activity
				if err := generateSkillLineCharts(history.Items); err != nil {
					log.Println("Skill line charts error:", err)
				}
				if err := generateActivityLineCharts(history.Items); err != nil {
					log.Println("Activity line charts error:", err)
				}
				// Create an overview page that contains both skills and activities charts
				if err := generateOverviewChart(history.Items); err != nil {
					log.Println("Overview chart error:", err)
				}

				pushBuild()
				log.Println("Courier Job Success")
			case <-c.ctx.Done():
				log.Println("Courier stopped")
				return
			}
		}
	}()
}

func (c *Courier) Stop() {
	c.cancel()
}

// pushBuild adds, commits, and pushes changes (build updates) to the repo.
func pushBuild() {
	addCmd := exec.Command("git", "add", ".")
	err := addCmd.Run()
	if err != nil {
		fmt.Println("Error adding files:", err)
		return
	}

	commitMsg := fmt.Sprintf("Build-%s", time.Now().Format("2006-01-02_15-04-05"))
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	err = commitCmd.Run()
	if err != nil {
		fmt.Println("Error committing changes:", err)
		return
	}

	pushCmd := exec.Command("git", "push", "origin", "main")
	err = pushCmd.Run()
	if err != nil {
		fmt.Println("Error pushing commit:", err)
		return
	}

	fmt.Println("Commit pushed successfully!")
}

// sanitizeFileName creates a safe file name from a given string.
func sanitizeFileName(name string) string {
	// Convert to lowercase and replace spaces with underscores.
	return strings.ReplaceAll(strings.ToLower(name), " ", "_")
}

// generateSkillLineCharts creates an HTML page (line chart) for each Skill (except "Overall") showing its XP progression.
func generateSkillLineCharts(history []osrsclient.PullAllItem) error {
	if len(history) == 0 {
		return nil
	}
	// Iterate over the skills found in the first history item.
	for _, skill := range history[0].Skills {
		if skill.Name == "Overall" {
			continue
		}
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: fmt.Sprintf("Skill: %s Progression", skill.Name)}),
			charts.WithXAxisOpts(opts.XAxis{Type: "category"}),
		)
		var timestamps []string
		var xpData []opts.LineData
		// For each polling item, extract the timestamp and the corresponding XP for this skill.
		for _, item := range history {
			timestamps = append(timestamps, item.TimeStamp.Format("2006-01-02 15:04"))
			var xp int32
			for _, s := range item.Skills {
				if s.Name == skill.Name {
					xp = s.XP
					break
				}
			}
			xpData = append(xpData, opts.LineData{Value: xp})
		}
		var istrue bool = true
		line.SetXAxis(timestamps).
			AddSeries(skill.Name, xpData).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: &istrue}))

		// Create an HTML file per skill in the "serve" directory.
		fileName := fmt.Sprintf("serve/skill_%s.html", sanitizeFileName(skill.Name))
		f, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := line.Render(f); err != nil {
			return err
		}
	}
	return nil
}

// generateActivityLineCharts creates an HTML page (line chart) for each Activity showing its Score progression.
func generateActivityLineCharts(history []osrsclient.PullAllItem) error {
	if len(history) == 0 {
		return nil
	}
	// Iterate over the activities from the first history item.
	for _, activity := range history[0].Acitivites {
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: fmt.Sprintf("Activity: %s Progression", activity.Name)}),
			charts.WithXAxisOpts(opts.XAxis{Type: "category"}),
		)
		var timestamps []string
		var scoreData []opts.LineData
		// For each polling item, extract the timestamp and the corresponding score for this activity.
		for _, item := range history {
			timestamps = append(timestamps, item.TimeStamp.Format("2006-01-02 15:04"))
			var score int32
			for _, a := range item.Acitivites {
				if a.Name == activity.Name {
					score = a.Score
					break
				}
			}
			scoreData = append(scoreData, opts.LineData{Value: score})
		}
		var istrue bool = true
		line.SetXAxis(timestamps).
			AddSeries(activity.Name, scoreData).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: &istrue}))

		// Create an HTML file per activity in the "serve" directory.
		fileName := fmt.Sprintf("serve/%s.html", sanitizeFileName(activity.Name))
		f, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := line.Render(f); err != nil {
			return err
		}
	}
	return nil
}

// generateOverviewChart creates an HTML page with an overview of skills and activities progression.
// It uses a Page container to display two charts on one page.
func generateOverviewChart(history []osrsclient.PullAllItem) error {
	if len(history) == 0 {
		return nil
	}

	// Create a combined skills overview chart.
	skillsLine := charts.NewLine()
	skillsLine.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Skills Overview"}),
		charts.WithXAxisOpts(opts.XAxis{Type: "category"}),
	)
	var timestamps []string
	for _, item := range history {
		timestamps = append(timestamps, item.TimeStamp.Format("2006-01-02 15:04"))
	}
	// Add a series for each skill (except "Overall")
	for _, skill := range history[0].Skills {
		if skill.Name == "Overall" {
			continue
		}
		var xpData []opts.LineData
		for _, item := range history {
			var xp int32
			for _, s := range item.Skills {
				if s.Name == skill.Name {
					xp = s.XP
					break
				}
			}
			xpData = append(xpData, opts.LineData{Value: xp})
		}
		var istrue bool = true
		skillsLine.AddSeries(skill.Name, xpData).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: &istrue}))
	}
	skillsLine.SetXAxis(timestamps)

	// Create a combined activities overview chart.
	activitiesLine := charts.NewLine()
	activitiesLine.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Activities Overview"}),
		charts.WithXAxisOpts(opts.XAxis{Type: "category"}),
	)
	// Add a series for each activity.
	for _, activity := range history[0].Acitivites {
		var scoreData []opts.LineData
		for _, item := range history {
			var score int32
			for _, a := range item.Acitivites {
				if a.Name == activity.Name {
					score = a.Score
					break
				}
			}
			scoreData = append(scoreData, opts.LineData{Value: score})
		}
		var istrue bool = true
		activitiesLine.AddSeries(activity.Name, scoreData).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: &istrue}))
	}
	activitiesLine.SetXAxis(timestamps)

	// Use a Page container to put both charts on one HTML page.
	page := components.NewPage()
	page.AddCharts(skillsLine, activitiesLine)

	f, err := os.Create("serve/overview.html")
	if err != nil {
		return err
	}
	defer f.Close()
	return page.Render(f)
}
