package builder

import (
	"fmt"
	"html/template"
	"os"
	"strconv"

	"github.com/qolors/gosrs/internal/osrsclient"
)

func BuildWithCharts(b []byte, skills []osrsclient.Skill, charts [][]byte) error {

	// Convert bytes to string and mark as safe HTML.
	safeHTML := template.HTML(string(b))

	chartHtmls := make([]template.HTML, len(charts))

	for _, chartbytes := range charts {

		chartHtml := template.HTML(string(chartbytes))

		chartHtmls = append(chartHtmls, chartHtml)

	}

	fmt.Print(chartHtmls[0])

	// Create a template with a placeholder for our HTML content.
	data := PageData{
		Content: safeHTML,
		Charts:  chartHtmls,
		Skills:  skills,
	}

	// Parse and execute the template
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		// optional: a helper to format numbers if needed
		"formatInt": func(i interface{}) string {
			switch v := i.(type) {
			case int16:
				return strconv.Itoa(int(v))
			case int32:
				return strconv.Itoa(int(v))
			default:
				return ""
			}
		},
	}).Parse(templateHTML)
	if err != nil {
		return err
	}

	fileName := "serve/overview.html"
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	return nil
}

type PageData struct {
	Content template.HTML
	Charts  []template.HTML
	Skills  []osrsclient.Skill // Array of skills to be displayed
}

var templateHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Skill Statistics</title>
  <style>
    body {
      font-family: Arial, sans-serif;
      margin: 20px;
      background-color: #f7f7f7;
    }
    #chart {
      margin-bottom: 30px;
    }

    /* Container for all cards */
    .card-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
      gap: 20px;
      margin-top: 20px;
    }

    /* Individual skill cards */
    .skill-card {
      background-color: #fff;
      border-radius: 8px;
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
      padding: 20px;
    }

    .skill-card h2 {
      margin: 0 0 10px 0;
      font-size: 1.2rem;
    }

    .skill-card p {
      margin: 8px 0;
      line-height: 1.4;
    }

    /* Headline styling */
    h1 {
      margin-top: 40px;
      font-size: 1.8rem;
    }
  </style>
</head>
<body>
  <!-- Provided chart content -->
  <div id="chart">
    {{.Content}}
  </div>

  <h1>Skill Statistics</h1>

  <div class="card-container">
    {{range $i, $s := .Skills}}
      <div class="skill-card">
        <h2>{{$s.Name}}</h2>
        <p><strong>Rank:</strong> {{$s.Rank}}</p>
        <p><strong>Level:</strong> {{$s.Level}}</p>
        <p><strong>XP:</strong> {{$s.XP}}</p>
		{{index $.Charts $i}}
      </div>
    {{end}}
  </div>
</body>
</html>
`
