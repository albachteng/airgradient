package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/albachteng/airgradient/internal/db"
)

const usage = `Usage:
  placement set <label> [description]   Tag where the monitor is now
  placement list                        Show all placement history
  placement current                     Show the active placement
`

func main() {
	godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	dbPath := envOr("DB_PATH", "./airgradient.db")
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	switch os.Args[1] {
	case "set":
		cmdSet(database)
	case "list":
		cmdList(database)
	case "current":
		cmdCurrent(database)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}

func cmdSet(database *db.DB) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: placement set <label> [description]")
		os.Exit(1)
	}
	label := os.Args[2]
	description := ""
	if len(os.Args) >= 4 {
		description = strings.Join(os.Args[3:], " ")
	}
	if err := database.SetPlacement(label, description); err != nil {
		log.Fatalf("set placement: %v", err)
	}
	fmt.Printf("placement set to %q\n", label)
}

func cmdList(database *db.DB) {
	placements, err := database.ListPlacements()
	if err != nil {
		log.Fatalf("list placements: %v", err)
	}
	if len(placements) == 0 {
		fmt.Println("no placements recorded")
		return
	}
	fmt.Printf("%-4s  %-20s  %-26s  %-26s  %s\n", "ID", "Label", "Started", "Ended", "Description")
	fmt.Println(strings.Repeat("-", 100))
	for _, p := range placements {
		ended := "(current)"
		if p.EndedAt != nil {
			ended = *p.EndedAt
		}
		fmt.Printf("%-4d  %-20s  %-26s  %-26s  %s\n", p.ID, p.Label, p.StartedAt, ended, p.Description)
	}
}

func cmdCurrent(database *db.DB) {
	p, err := database.CurrentPlacement()
	if err != nil {
		log.Fatalf("current placement: %v", err)
	}
	if p == nil {
		fmt.Println("no active placement — use: placement set <label>")
		return
	}
	desc := ""
	if p.Description != "" {
		desc = "  (" + p.Description + ")"
	}
	fmt.Printf("current placement: %s%s  [since %s]\n", p.Label, desc, p.StartedAt)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
