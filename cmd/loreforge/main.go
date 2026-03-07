package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"loreforge/internal/config"
	"loreforge/internal/core"
	"loreforge/internal/scheduler"
	"loreforge/internal/universe"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "validate":
		validateCmd(os.Args[2:])
	case "generate":
		generateCmd(os.Args[2:])
	case "episode":
		episodeCmd(os.Args[2:])
	case "universe":
		universeCmd(os.Args[2:])
	case "scheduler":
		schedulerCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config/config.yaml", "path to config yaml")
	_ = fs.Parse(args)
	cfg := loadConfigOrExit(*configPath)
	eng, err := core.New(cfg)
	must(err)
	rec, err := eng.GenerateOnce(context.Background(), "")
	must(err)
	fmt.Printf("run complete: episode=%s state=%s type=%s\n", rec.Manifest.EpisodeID, rec.Manifest.State, rec.Manifest.OutputType)
}

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config/config.yaml", "path to config yaml")
	_ = fs.Parse(args)
	cfg := loadConfigOrExit(*configPath)
	eng, err := core.New(cfg)
	must(err)
	must(eng.ValidateUniverse())
	fmt.Println("validate ok")
}

func generateCmd(args []string) {
	if len(args) == 0 || args[0] != "once" {
		fmt.Fprintln(os.Stderr, "usage: loreforge generate once --agent text --config ./config.yaml")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("generate once", flag.ExitOnError)
	agent := fs.String("agent", "", "agent type: text|video")
	configPath := fs.String("config", "./universes/config/config.yaml", "path to config yaml")
	_ = fs.Parse(args[1:])
	cfg := loadConfigOrExit(*configPath)
	eng, err := core.New(cfg)
	must(err)
	rec, err := eng.GenerateOnce(context.Background(), *agent)
	must(err)
	fmt.Printf("generated: episode=%s type=%s\n", rec.Manifest.EpisodeID, rec.Manifest.OutputType)
}

func episodeCmd(args []string) {
	if len(args) < 2 || args[0] != "show" {
		fmt.Fprintln(os.Stderr, "usage: loreforge episode show <id> --config ./config.yaml")
		os.Exit(1)
	}
	epID := args[1]
	fs := flag.NewFlagSet("episode show", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config/config.yaml", "path to config yaml")
	_ = fs.Parse(args[2:])
	cfg := loadConfigOrExit(*configPath)
	eng, err := core.New(cfg)
	must(err)
	path, manifest, err := eng.ShowEpisode(epID)
	must(err)
	b, _ := json.MarshalIndent(manifest, "", "  ")
	fmt.Printf("episode path: %s\n%s\n", path, string(b))
}

func universeCmd(args []string) {
	if len(args) < 2 || args[0] != "lint" {
		fmt.Fprintln(os.Stderr, "usage: loreforge universe lint ./universe")
		os.Exit(1)
	}
	path := args[1]
	_, err := universe.Load(path)
	must(err)
	fmt.Println("universe lint ok")
}

func schedulerCmd(args []string) {
	if len(args) == 0 || args[0] != "next-run" {
		fmt.Fprintln(os.Stderr, "usage: loreforge scheduler next-run --config ./config.yaml")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("scheduler next-run", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config/config.yaml", "path to config yaml")
	_ = fs.Parse(args[1:])
	cfg := loadConfigOrExit(*configPath)
	minInt, _ := time.ParseDuration(cfg.Scheduler.MinInterval)
	maxInt, _ := time.ParseDuration(cfg.Scheduler.MaxInterval)
	fixInt, _ := time.ParseDuration(cfg.Scheduler.FixedInterval)
	sch, err := scheduler.New(scheduler.Config{
		Mode:          cfg.Scheduler.Mode,
		MinInterval:   minInt,
		MaxInterval:   maxInt,
		FixedInterval: fixInt,
		Seed:          cfg.Scheduler.Seed,
		Timezone:      cfg.Scheduler.Timezone,
	})
	must(err)
	next, err := sch.NextRun(time.Now())
	must(err)
	fmt.Printf("next run: %s\n", next.Format(time.RFC3339))
}

func loadConfigOrExit(path string) config.Config {
	cfg, err := config.Load(path)
	must(err)
	return cfg
}

func usage() {
	fmt.Println(strings.TrimSpace(`
Usage:
  loreforge run --config ./config.yaml
  loreforge validate --config ./config.yaml
  loreforge generate once --agent text --config ./config.yaml
  loreforge episode show <id> --config ./config.yaml
  loreforge universe lint ./universe
  loreforge scheduler next-run --config ./config.yaml
`))
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
