package services

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
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
				latestSkills := history.Items[0].Skills
				if err := generateSkillChart(latestSkills); err != nil {
					log.Println("Skill chart error:", err)
				}
				if err := generateSkillLineChart(history.Items); err != nil {
					log.Println("Line chart error:", err)
				}
			case <-c.ctx.Done():
				log.Println("Courier stopped")
				return
			}
		}
	}()
}

func generateSkillChart(skills []osrsclient.Skill) error {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Current Skills XP Snapshot"}),
	)

	var names []string
	var data []opts.BarData
	for _, skill := range skills {
		if skill.Name != "Overall" {
			names = append(names, skill.Name)
			data = append(data, opts.BarData{Value: skill.XP})
		}
	}

	bar.SetXAxis(names).AddSeries("XP", data)

	f, err := os.Create("skills_snapshot.html")
	if err != nil {
		return err
	}
	defer f.Close()
	return bar.Render(f)
}

func generateSkillLineChart(history []osrsclient.PullAllItem) error {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Skill XP Trends (Last 100 Polls)"}),
		charts.WithLegendOpts(opts.Legend{}),
	)

	var timestamps []string
	for _, item := range history {
		timestamps = append(timestamps, item.TimeStamp.Format("2006-01-02 15:04"))
	}

	skillSeries := make(map[string][]opts.LineData)
	for _, skill := range history[0].Skills {
		if skill.Name == "Overall" {
			continue
		}
		skillSeries[skill.Name] = []opts.LineData{}
	}

	for _, item := range history {
		for _, skill := range item.Skills {
			if skill.Name == "Overall" {
				continue
			}
			skillSeries[skill.Name] = append(skillSeries[skill.Name], opts.LineData{Value: skill.XP})
		}
	}

	line.SetXAxis(timestamps)
	for skillName, data := range skillSeries {
		line.AddSeries(skillName, data)
	}

	f, err := os.Create("skills_trends.html")
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

func (c *Courier) Stop() {
	c.cancel()
}
