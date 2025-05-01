package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/enolgor/freshrss-ai/app"
	"github.com/enolgor/freshrss-ai/config"
	"github.com/enolgor/freshrss-ai/data"
	_ "modernc.org/sqlite" // Import the SQLite driver
)

var date string
var configFile string
var subcommand string
var cfg *config.Configuration
var upload bool

func init() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: news-report <feeds|report> [--config=...] [--date=...]")
		os.Exit(1)
	}
	subcommand = os.Args[1]
	args := os.Args[2:]
	feedsFlagSet := flag.NewFlagSet("feeds", flag.ExitOnError)
	reportFlagSet := flag.NewFlagSet("report", flag.ExitOnError)
	feedsFlagSet.StringVar(&configFile, "config", "./config.yaml", "Path to config file")
	reportFlagSet.StringVar(&configFile, "config", "./config.yaml", "Path to config file")
	reportFlagSet.StringVar(&date, "date", "", "Date argument in YYYY-MM-DD format")
	reportFlagSet.BoolVar(&upload, "upload", false, "Upload report to github")

	switch subcommand {
	case "feeds":
		feedsFlagSet.Parse(args)
	case "report":
		reportFlagSet.Parse(args)
	default:
		fmt.Fprintln(os.Stderr, "expected 'feeds' or 'report' subcommand")
		os.Exit(1)
	}

	config.EnsureConfigFile(configFile)
	var err error
	cfg, err = config.ParseConfig(configFile)
	if err != nil {
		log.Fatalf("could not parse config file: %s\n", err.Error())
	}
}

func main() {

	switch subcommand {
	case "feeds":
		runFeeds()
	case "report":
		runReport()
	default:
		fmt.Fprintln(os.Stderr, "expected 'feeds' or 'report' subcommand")
		os.Exit(1)
	}

}

func runFeeds() {
	if cfg.DbFile == "" {
		log.Fatalf("dbFile not found in config file")
	}
	db, err := sql.Open("sqlite", cfg.DbFile)
	if err != nil {
		log.Fatalf("could not open db file: %s\n", err.Error())
	}
	defer db.Close()
	queries := data.New(db)
	ctx := context.Background()
	feeds, err := queries.GetFeeds(ctx)
	if err != nil {
		log.Fatalf("could not get feeds: %s\n", err.Error())
	}
	for _, feed := range feeds {
		fmt.Printf("%d: %s\n", feed.ID, feed.Name)
	}
}

func runReport() {
	if upload {
		if cfg.Github.Repo == "" || cfg.Github.Branch == "" || cfg.Github.Pat == "" {
			log.Fatalf("github repo, branch and pat not found in config file")
		}
	}
	app := app.NewApplication(cfg)
	var refDate time.Time
	var err error
	if date == "" {
		refDate = time.Now().AddDate(0, 0, -1)
	} else {
		if refDate, err = time.Parse("2006-01-02", date); err != nil {
			log.Fatalf("could not parse date: %s\n", err.Error())
		}
	}
	newsSummary, err := app.NewsSummaryForDay(refDate)
	if err != nil {
		log.Fatal(err)
	}
	aiPrompt, err := newsSummary.GenerateAiPrompt()
	if err != nil {
		log.Fatal(err)
	}
	newsSummary.AiSummary, err = app.OpenAiRequest(aiPrompt)
	if err != nil {
		log.Fatal(err)
	}
	if upload {
		buf := new(bytes.Buffer)
		if err := newsSummary.GenerateReport(buf); err != nil {
			log.Fatal(err)
		}
		filename := path.Join(cfg.Github.Path, fmt.Sprintf("%s.md", refDate.Format("2006-01-02")))
		if err := app.UploadReport(buf.Bytes(), filename); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := newsSummary.GenerateReport(os.Stdout); err != nil {
			log.Fatal(err)
		}
	}
}
