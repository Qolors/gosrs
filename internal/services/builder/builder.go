package builder

import (
	"html/template"
	"os"
	"strconv"

	"github.com/qolors/gosrs/internal/osrsclient"
)

func BuildWithCharts(b []byte, skills []osrsclient.Skill) error {

	// Convert bytes to string and mark as safe HTML.
	safeHTML := template.HTML(string(b))

	// Create a template with a placeholder for our HTML content.
	data := PageData{
		Content: safeHTML,
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
	Content template.HTML      // The provided chart as safe HTML
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
		}
		#chart {
			margin-bottom: 30px;
		}
		table {
			width: 100%;
			border-collapse: collapse;
			margin-bottom: 20px;
		}
		th, td {
			border: 1px solid #ccc;
			padding: 10px;
			text-align: left;
		}
		th {
			background-color: #f2f2f2;
		}
	</style>
</head>
<body>
	<!-- Provided chart content -->
	<div id="chart">
		{{.Content}}
	</div>
	<h1>Skill Statistics</h1>
	<table>
		<thead>
			<tr>
				<th>ID</th>
				<th>Name</th>
				<th>Rank</th>
				<th>Level</th>
				<th>XP</th>
			</tr>
		</thead>
		<tbody>
			{{range .Skills}}
			<tr>
				<td>{{.ID}}</td>
				<td>{{.Name}}</td>
				<td>{{.Rank}}</td>
				<td>{{.Level}}</td>
				<td>{{.XP}}</td>
			</tr>
			{{end}}
		</tbody>
	</table>
</body>
</html>
`
