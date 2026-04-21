package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"ali-tracker/src/aliexpress"
	"ali-tracker/src/telegram"
	"ali-tracker/src/tracker"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	rapidAPIKey := mustEnv("RAPIDAPI_KEY")
	rapidAPIHost := envOrDefault("RAPIDAPI_HOST", "aliexpress-datahub.p.rapidapi.com")
	telegramToken := mustEnv("TELEGRAM_TOKEN")
	telegramChatID := mustEnv("TELEGRAM_CHAT_ID")

	region := envOrDefault("ALI_REGION", "BR")
	locale := envOrDefault("ALI_LOCALE", "pt_BR")
	currency := envOrDefault("ALI_CURRENCY", "BRL")
	topN := envIntOrDefault("TRACKER_TOP_N", 5)
	keywords := envListOrDefault("ALI_SEARCH_TERMS", []string{"kit xeon x99", "memoria ram", "placa de video"})

	client := aliexpress.NewClient(rapidAPIKey, rapidAPIHost)
	bot := telegram.NewBot(telegramToken)
	worker := tracker.NewWorker(client, bot, telegramChatID, topN, region, locale, currency)

	enableScheduler := envBoolOrDefault("TRACKER_ENABLE_SCHEDULER", false)
	if !enableScheduler {
		if err := runCycle(worker, keywords); err != nil {
			log.Fatalf("tracker run failed: %v", err)
		}
		log.Println("tracker single run completed successfully")
		return
	}

	runOnStart := envBoolOrDefault("TRACKER_RUN_ON_START", true)
	runTimes := envListOrDefault("TRACKER_RUN_TIMES", []string{"09:00", "15:00", "21:00"})
	location := mustLocation(envOrDefault("TRACKER_TIMEZONE", "America/Sao_Paulo"))

	if runOnStart {
		if err := runCycle(worker, keywords); err != nil {
			log.Printf("tracker startup run failed: %v", err)
		}
	}

	if len(runTimes) == 0 {
		log.Println("no TRACKER_RUN_TIMES configured; finishing after startup run")
		return
	}

	timesOfDay, err := parseDailyTimes(runTimes)
	if err != nil {
		log.Fatalf("invalid TRACKER_RUN_TIMES: %v", err)
	}

	log.Printf("scheduler enabled (%s), runs at: %s", location.String(), strings.Join(runTimes, ", "))
	for {
		next := nextRunTime(time.Now().In(location), timesOfDay, location)
		sleep := time.Until(next)
		log.Printf("next run scheduled for %s", next.Format(time.RFC3339))
		time.Sleep(sleep)
		if err := runCycle(worker, keywords); err != nil {
			log.Printf("tracker scheduled run failed: %v", err)
		}
	}
}

func runCycle(worker *tracker.Worker, keywords []string) error {
	if err := worker.Run(context.Background(), keywords); err != nil {
		return err
	}
	return nil
}

func parseDailyTimes(values []string) ([]time.Duration, error) {
	out := make([]time.Duration, 0, len(values))
	seen := make(map[time.Duration]struct{}, len(values))

	for _, raw := range values {
		parts := strings.Split(strings.TrimSpace(raw), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid time %q, expected HH:MM", raw)
		}

		hour, err := strconv.Atoi(parts[0])
		if err != nil || hour < 0 || hour > 23 {
			return nil, fmt.Errorf("invalid hour in %q", raw)
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil || minute < 0 || minute > 59 {
			return nil, fmt.Errorf("invalid minute in %q", raw)
		}

		d := time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
		if _, exists := seen[d]; exists {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func nextRunTime(now time.Time, daily []time.Duration, location *time.Location) time.Time {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	for _, offset := range daily {
		candidate := startOfDay.Add(offset)
		if candidate.After(now) {
			return candidate
		}
	}
	return startOfDay.Add(24 * time.Hour).Add(daily[0])
}

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(strings.TrimSpace(name))
	if err != nil {
		log.Fatalf("invalid TRACKER_TIMEZONE %q: %v", name, err)
	}
	return loc
}

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("missing required env variable: %s", key)
	}
	return value
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envListOrDefault(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	if len(items) == 0 {
		return fallback
	}
	return items
}
