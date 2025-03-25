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
				// Generate per-skill and per-activity line charts.
				//if err := generateSkillLineCharts(history.Items); err != nil {
				//	log.Println("Skill line charts error:", err)
				//}
				//if err := generateActivityLineCharts(history.Items); err != nil {
				//	log.Println("Activity line charts error:", err)
				//}
				// Generate the overview page that contains:
				// 1. A scatter chart of the latest ranks.
				// 2. A candle (KLine) chart of the "Overall" XP progression.
				if err := generateOverviewPage(history.Items); err != nil {
					log.Println("Overview page error:", err)
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

// pushBuild adds, commits, and pushes the generated changes to your repository.
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

// sanitizeFileName converts a string into a safe filename (lowercase, underscores).
func sanitizeFileName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "_")
}

// generateSkillLineCharts creates an HTML line chart for each skill (except "Overall").
func generateSkillLineCharts(history []osrsclient.PullAllItem) error {
	if len(history) == 0 {
		return nil
	}
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

// generateActivityLineCharts creates an HTML line chart for each activity.
func generateActivityLineCharts(history []osrsclient.PullAllItem) error {
	if len(history) == 0 {
		return nil
	}
	for _, activity := range history[0].Acitivites {
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: fmt.Sprintf("Activity: %s Progression", activity.Name)}),
			charts.WithXAxisOpts(opts.XAxis{Type: "category"}),
		)
		var timestamps []string
		var scoreData []opts.LineData
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

		fileName := fmt.Sprintf("serve/activity_%s.html", sanitizeFileName(activity.Name))
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

// generateOverviewPage creates an HTML page combining the overview scatter and candle charts.
func generateOverviewPage(history []osrsclient.PullAllItem) error {
	scatterChart, err := getOverviewScatterChart(history)
	if err != nil {
		return err
	}
	candleChart, err := getSkillCandleChart(history)
	if err != nil {
		return err
	}
	page := components.NewPage()
	page.AddCharts(scatterChart, candleChart)

	f, err := os.Create("serve/overview.html")
	if err != nil {
		return err
	}
	defer f.Close()
	return page.Render(f)
}

// getOverviewScatterChart returns a scatter chart that plots the latest skills and Activites ranks.
// X axis: names (categories), Y axis: rank values.
func getOverviewScatterChart(history []osrsclient.PullAllItem) (*charts.Scatter, error) {
	if len(history) == 0 {
		return nil, nil
	}
	latest := history[0]
	var skillsData []opts.ScatterData
	var activitiesData []opts.ScatterData
	var categories []string

	// Add skills (excluding "Overall")
	for _, skill := range latest.Skills {
		if skill.Name == "Overall" {
			continue
		}
		skillsData = append(skillsData, opts.ScatterData{
			Value: []interface{}{skill.Name, skill.Rank},
		})
		categories = append(categories, skill.Name)
	}
	// Add activities
	for _, activity := range latest.Acitivites {
		activitiesData = append(activitiesData, opts.ScatterData{
			Value: []interface{}{activity.Name, activity.Rank},
		})
		categories = append(categories, activity.Name)
	}

	scatterChart := charts.NewScatter()
	scatterChart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Overview: Latest Ranks"}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			Data: categories,
		}),
	)
	scatterChart.AddSeries("Skills", skillsData)
	scatterChart.AddSeries("Activities", activitiesData)
	return scatterChart, nil
}

// getSkillCandleChart returns a KLine (candlestick) chart of the "Overall" skill's XP progression.
// Each candle uses two consecutive history items.
func getSkillCandleChart(history []osrsclient.PullAllItem) (*charts.Kline, error) {
	if len(history) < 2 {
		return nil, nil
	}
	var timestamps []string
	var candleData []opts.KlineData

	for i := 1; i < len(history); i++ {
		prevItem := history[i-1]
		currItem := history[i]
		var prevXP, currXP int32
		for _, s := range prevItem.Skills {
			if s.Name == "Overall" {
				prevXP = s.XP
				break
			}
		}
		for _, s := range currItem.Skills {
			if s.Name == "Overall" {
				currXP = s.XP
				break
			}
		}
		open := prevXP
		close := currXP
		low := open
		high := close
		if open > close {
			low = close
			high = open
		} else {
			low = open
			high = close
		}
		timestamps = append(timestamps, currItem.TimeStamp.Format("2006-01-02 15:04"))
		candleData = append(candleData, opts.KlineData{
			Value: []interface{}{open, close, low, high},
		})
	}

	var istrue bool = true
	kline := charts.NewKLine()
	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Overall XP Candle Chart"}),
		charts.WithXAxisOpts(opts.XAxis{SplitNumber: 20}),
		charts.WithYAxisOpts(opts.YAxis{Scale: &istrue}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
	)
	kline.SetXAxis(timestamps).AddSeries("Overall XP", candleData)
	return kline, nil
}
