package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const version = "0.1.0"

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tasks.db"
	}

	allowed := parseAllowedUsers(os.Getenv("ALLOWED_USERS"))

	store, err := newStore(dbPath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	bot, err := newBot(token, store, allowed)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("telestas %s up (allowed private users: %d)", version, len(allowed))
	bot.Run(ctx)
	log.Print("telestas shutdown")
}

func parseAllowedUsers(raw string) map[int64]struct{} {
	out := map[int64]struct{}{}
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Printf("ALLOWED_USERS: skipping invalid id %q: %v", s, err)
			continue
		}
		out[id] = struct{}{}
	}
	return out
}
