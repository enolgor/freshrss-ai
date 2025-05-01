package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/enolgor/freshrss-ai/config"
	"github.com/enolgor/freshrss-ai/data"
	"github.com/enolgor/freshrss-ai/models"
	"github.com/k3a/html2text"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	_ "modernc.org/sqlite" // Import the SQLite driver
)

type Application struct {
	loc *time.Location
	cfg *config.Configuration
}

func NewApplication(cfg *config.Configuration) *Application {
	if cfg.DbFile == "" {
		log.Fatalf("dbFile not found in config file")
	}
	if cfg.FeedIds == nil || len(cfg.FeedIds) == 0 {
		log.Fatalf("feedIds not found in config file")
	}
	if cfg.OpenAi.ApiKey == "" {
		log.Fatalf("open-ai-api-key not found in config file")
	}
	if cfg.OpenAi.Model == "" {
		log.Fatalf("open-ai-api-key not found in config file")
	}
	loc, err := time.LoadLocation(cfg.Location)
	if err != nil {
		log.Fatalf("could not load location: %s\n", err.Error())
	}
	return &Application{
		loc: loc,
		cfg: cfg,
	}
}

func (app *Application) NewsSummaryForDay(date time.Time) (*NewsSummary, error) {
	dateLoc := date.In(app.loc)
	fromDate := time.Date(
		dateLoc.Year(),
		dateLoc.Month(),
		dateLoc.Day(),
		0, 0, 0, 0,
		app.loc,
	)
	toDate := fromDate.AddDate(0, 0, 1)
	db, err := sql.Open("sqlite", app.cfg.DbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	queries := data.New(db)
	ctx := context.Background()
	allFeeds, err := queries.GetFeeds(ctx)
	if err != nil {
		return nil, err
	}
	feeds := []models.Feed{}
	for i := range allFeeds {
		if !slices.Contains(app.cfg.FeedIds, allFeeds[i].ID) {
			continue
		}
		entries, err := queries.GetEntries(ctx, data.GetEntriesParams{
			FeedID:   allFeeds[i].ID,
			FromDate: fromDate.Unix(),
			ToDate:   toDate.Unix(),
		})
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			continue
		}
		feed := models.Feed{
			Name:    allFeeds[i].Name,
			Entries: make([]models.Entry, len(entries)),
		}
		feed.Entries = make([]models.Entry, len(entries))
		for j, entry := range entries {
			feed.Entries[j].Title = entry.Title
			feed.Entries[j].Link = entry.Link
			feed.Entries[j].Content = parseContent(entry.Content)
		}
		feeds = append(feeds, feed)
	}
	return &NewsSummary{
		Date:  spanishDate(date),
		Feeds: feeds,
	}, nil
}

func spanishDate(date time.Time) string {
	weekDay := date.Weekday()
	month := date.Month()
	day := date.Day()
	year := date.Year()
	weekDays := []string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}
	months := []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
	return fmt.Sprintf("%s %d de %s de %d", weekDays[weekDay], day, months[month-1], year)
}

func parseContent(content *string) string {
	if content == nil {
		return ""
	}
	parsed := html2text.HTML2Text(*content)
	parsed = strings.ReplaceAll(parsed, "#", "+")
	return parsed
}

func (app *Application) OpenAiRequest(aiPrompt string) (string, error) {
	client := openai.NewClient(
		option.WithAPIKey(app.cfg.OpenAi.ApiKey),
	)
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(aiPrompt),
		},
		Model: shared.ChatModel(app.cfg.OpenAi.Model),
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

type Committer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type RequestPayload struct {
	Message   string    `json:"message"`
	Content   string    `json:"content"`
	Branch    string    `json:"branch"`
	Committer Committer `json:"committer"`
	Author    Committer `json:"author"`
}

func (app *Application) UploadReport(content []byte, fileName string) error {
	contentEncoded := base64.StdEncoding.EncodeToString(content)

	payload := RequestPayload{
		Message: "Add " + fileName,
		Content: contentEncoded,
		Branch:  app.cfg.Github.Branch,
		Committer: Committer{
			Name:  "News Bot",
			Email: "news@enolgor.es",
		},
		Author: Committer{
			Name:  "News Bot",
			Email: "news@enolgor.es",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", app.cfg.Github.Repo, fileName)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+app.cfg.Github.Pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ File '%s' uploaded successfully!\n", fileName)
	} else {
		fmt.Printf("❌ GitHub API error: %s\n", resp.Status)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Println("Response:", buf.String())
		os.Exit(1)
	}
	return nil
}
