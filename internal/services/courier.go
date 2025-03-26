package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/qolors/gosrs/internal/osrsclient"
	"github.com/qolors/gosrs/internal/services/builder"
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
				if err := generateOverviewPage(history.Items); err != nil {
					log.Println("Overview page error:", err)
				}
				if err = generateCharacterBuildPage(history.Items); err != nil {
					log.Println("Character page error:", err)
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

		fileName := fmt.Sprintf("serve/%s.html", skill.Name)
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

		fileName := fmt.Sprintf("serve/%s.html", activity.Name)
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

	candleChart := getSkillCandleChart(history)

	var buffer bytes.Buffer

	candleChart.Render(&buffer)

	return builder.BuildWithCharts(buffer.Bytes(), history[0].Skills)
}

func generateCharacterBuildPage(history []osrsclient.PullAllItem) error {

	pieChart := getOverviewPieChart(history[0])

	page := components.NewPage()
	page.AddCharts(pieChart)

	f, err := os.Create("serve/character.html")
	if err != nil {
		return err
	}
	defer f.Close()
	return page.Render(f)
}

// getOverviewScatterChart returns a scatter chart that plots the latest skills and Activites ranks.
// X axis: names (categories), Y axis: rank values.
func getOverviewPieChart(latest osrsclient.PullAllItem) *charts.Pie {

	const actionWithEchartsInstance = `
		let currentIndex = -1;
		setInterval(function() {
		  const myChart = %MY_ECHARTS%;
		  var dataLen = myChart.getOption().series[0].data.length;
		  myChart.dispatchAction({
			type: 'downplay',
			seriesIndex: 0,
			dataIndex: currentIndex
		  });
		  currentIndex = (currentIndex + 1) % dataLen;
		  myChart.dispatchAction({
			type: 'highlight',
			seriesIndex: 0,
			dataIndex: currentIndex
		  });
		  myChart.dispatchAction({
			type: 'showTip',
			seriesIndex: 0,
			dataIndex: currentIndex
		  });
		}, 2000);
`

	pieData := make(map[string]int32, 4)

	pieData["Combat"] = 0
	pieData["Gathering"] = 0
	pieData["Production"] = 0
	pieData["Utility"] = 0

	pieItems := make([]opts.PieData, 0)

	for _, skill := range latest.Skills {
		switch name := skill.Name; name {
		case "Attack", "Defense", "Hitpoints", "Magic", "Prayer", "Ranged", "Strength":
			pieData["Combat"] += skill.XP
		case "Farming", "Fishing", "Hunter", "Mining", "Woodcutting":
			pieData["Gathering"] += skill.XP
		case "Cooking", "Crafting", "Fletching", "Herblore", "Runecraft", "Smithing":
			pieData["Production"] += skill.XP
		case "Agility", "Construction", "Firemaking", "Slayer", "Thieving":
			pieData["Utility"] += skill.XP

		}
	}

	pieItems = append(pieItems, opts.PieData{Name: "Combat", Value: pieData["Combat"]})
	pieItems = append(pieItems, opts.PieData{Name: "Gathering", Value: pieData["Gathering"]})
	pieItems = append(pieItems, opts.PieData{Name: "Production", Value: pieData["Production"]})
	pieItems = append(pieItems, opts.PieData{Name: "Utility", Value: pieData["Utility"]})

	pie := charts.NewPie()
	pie.AddJSFuncStrs(actionWithEchartsInstance)
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "An Okay Time Build Allocation",
			Right: "40%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "item",
			Formatter: "{a} <br/>{b} : {c}xp ({d}%)",
		}),
		charts.WithLegendOpts(opts.Legend{
			Left:   "left",
			Orient: "vertical",
		}),
	)

	pie.AddSeries("Experience Dist.", pieItems).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}: {c}",
			}),
			charts.WithPieChartOpts(opts.PieChart{
				Radius: []string{"55%"},
				Center: []string{"50%", "60%"},
			}),

			charts.WithEmphasisOpts(opts.Emphasis{
				ItemStyle: &opts.ItemStyle{
					ShadowBlur:    10,
					ShadowOffsetX: 0,
					ShadowColor:   "rgba(0, 0, 0, 0.5)",
				},
			}),
		)

	return pie
}

// getSkillCandleChart returns a KLine (candlestick) chart of the "Overall" skill's XP progression.
// Each candle uses two consecutive history items.
func getSkillCandleChart(history []osrsclient.PullAllItem) *charts.Bar {

	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Average XP in the last hour",
		}),
	)

	var xAxis []string
	var gainItems []opts.BarData

	// Ensure there are at least two history items to compare.
	if len(history) < 2 {
		return bar
	}

	// Load Eastern Time location.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.Local
	}

	// Define cutoff: one hour before the latest polled data.
	latestTime := history[0].TimeStamp
	cutoffTime := latestTime.Add(-time.Hour)

	// Iterate from newest (index 0) to second-to-last record.
	for i := 0; i < len(history)-1; i++ {
		// Break out if the current record is before the cutoff.
		if history[i].TimeStamp.Before(cutoffTime) {
			break
		}

		// If the next record is older than cutoff, skip this pair.
		if history[i+1].TimeStamp.Before(cutoffTime) {
			continue
		}

		// Calculate total XP for the current (newer) record.
		newerXP := 0
		for _, skill := range history[i].Skills {
			newerXP += int(skill.XP)
		}

		// Calculate total XP for the next (older) record.
		olderXP := 0
		for _, skill := range history[i+1].Skills {
			olderXP += int(skill.XP)
		}

		// Gain is computed as the difference between the newer and the older record.
		gain := newerXP - olderXP

		// Format the x-axis label to Eastern Standard Time.
		xAxis = append(xAxis, history[i].TimeStamp.In(loc).Format("3:04pm"))
		gainItems = append(gainItems, opts.BarData{Value: gain})
	}

	bar.SetXAxis(xAxis).
		AddSeries("XP Gain", gainItems).
		SetSeriesOptions(charts.WithMarkLineNameTypeItemOpts(
			opts.MarkLineNameTypeItem{Name: "Maximum", Type: "max"},
			opts.MarkLineNameTypeItem{Name: "Avg", Type: "average"},
		))
	return bar
}
