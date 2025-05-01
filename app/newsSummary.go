package app

import (
	"bytes"
	"html/template"
	"io"

	"github.com/enolgor/freshrss-ai/models"
)

const aiPrompt = `
You are a journalist summarizing the following raw news items into a single natural blog post.
Do **not** include any lists, bullet points, article titles, or raw URLs.
Your goal is to **paraphrase** and **summarize** these articles naturally.
Write it like a human blog post: fluent, cohesive, relaxed.
Divide the content into the following sections:
  ## Sucesos y noticias relevantes
  ## Política, economía y sociedad
  ## Cultura y entretenimiento
  ## Tecnología y ciencia
Jump directly into the sections (use ##), do not mention this is a blog post or a summary.
Use the information below as reference material only — do **not** echo it directly.

# News of {{ .Date }}

{{ range .Feeds }}
## {{ .Name }}

{{ range .Entries }}
### [{{ .Title }}]({{ .Link }})

{{ .Content }}

{{ end }}
{{ end }}
`

const report = `
# {{ .Date }}

{{ .AiSummary }}

# Fuentes
{{ range .Feeds }}
## {{ .Name }}
{{ range .Entries }}
- [{{ .Title }}]({{ .Link }})
{{ end }}
{{ end }}
`

type NewsSummary struct {
	Date      string
	Feeds     []models.Feed
	AiSummary string
}

func (ns *NewsSummary) GenerateAiPrompt() (string, error) {
	tmpl, err := template.New("ai-prompt").Parse(aiPrompt)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, ns); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (ns *NewsSummary) GenerateReport(w io.Writer) error {
	tmpl, err := template.New("report").Parse(report)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, ns)
}
