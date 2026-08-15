package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/albachteng/airgradient/internal/db"
	"github.com/albachteng/airgradient/internal/emailer"
	"github.com/albachteng/airgradient/internal/summary"
)

func main() {
	godotenv.Load()

	dbPath := envOr("DB_PATH", "./airgradient.db")
	period := envOr("SUMMARY_PERIOD", "week")

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC()
	var from time.Time
	switch period {
	case "month":
		from = now.AddDate(0, -1, 0)
	default:
		from = now.AddDate(0, 0, -7)
	}

	s, err := summary.Generate(database, from, now)
	if err != nil {
		log.Fatalf("generate summary: %v", err)
	}

	html, err := summary.RenderHTML(s)
	if err != nil {
		log.Fatalf("render HTML: %v", err)
	}

	cfg := emailer.Config{
		SMTPHost: envOr("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort: envInt("SMTP_PORT", 587),
		Username: requireEnv("SMTP_USERNAME"),
		Password: requireEnv("SMTP_PASSWORD"),
		From:     requireEnv("SMTP_USERNAME"),
		To:       []string{requireEnv("SUMMARY_TO")},
	}

	subject := fmt.Sprintf("Air Quality Summary: %s – %s",
		from.Format("Jan 2"),
		now.Format("Jan 2, 2006"),
	)

	e := emailer.New(cfg)
	if err := e.Send(subject, html); err != nil {
		log.Fatalf("send email: %v", err)
	}

	fmt.Printf("summary sent to %s\n", cfg.To[0])
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "required env var %s is not set\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
