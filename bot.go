package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	store   *Store
	allowed map[int64]struct{}
}

func newBot(token string, store *Store, allowed map[int64]struct{}) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("authenticated as @%s", api.Self.UserName)
	return &Bot{api: api, store: store, allowed: allowed}, nil
}

func (b *Bot) Run(ctx context.Context) {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 30
	updates := b.api.GetUpdatesChan(cfg)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return
		case upd, ok := <-updates:
			if !ok {
				return
			}
			if upd.Message == nil || !upd.Message.IsCommand() {
				continue
			}
			b.handle(upd.Message)
		}
	}
}

// isAllowed: anyone can use the bot inside a group/supergroup; in private
// chats only users whose ID is in ALLOWED_USERS may use it.
func (b *Bot) isAllowed(msg *tgbotapi.Message) bool {
	if msg.Chat == nil {
		return false
	}
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		return true
	}
	if msg.From == nil {
		return false
	}
	_, ok := b.allowed[msg.From.ID]
	return ok
}

func (b *Bot) handle(msg *tgbotapi.Message) {
	if !b.isAllowed(msg) {
		log.Printf("denied: user=%d chat=%d cmd=%s", msg.From.ID, msg.Chat.ID, msg.Command())
		return
	}
	args := strings.TrimSpace(msg.CommandArguments())
	switch msg.Command() {
	case "start", "help":
		b.reply(msg, helpText)
	case "version":
		b.reply(msg, "telestas "+version)
	case "new":
		b.cmdNew(msg, args, false)
	case "newp":
		b.cmdNew(msg, args, true)
	case "list":
		b.cmdList(msg)
	case "edit":
		b.cmdEdit(msg, args)
	case "priority":
		b.cmdPriority(msg, args)
	case "done":
		b.cmdDone(msg, args)
	case "delete", "del":
		b.cmdDelete(msg, args)
	default:
		b.reply(msg, "Comando no reconocido. Mira /help.")
	}
}

const helpText = `Comandos:
/new <texto> — nueva tarea
/newp <texto> — nueva tarea prioritaria 🔥
/list — listar tareas abiertas
/edit <id> <texto> — modificar el texto
/priority <id> — alternar prioridad
/done <id> — cerrar tarea
/delete <id> — borrar tarea
/version — versión del bot`

func (b *Bot) reply(msg *tgbotapi.Message, text string) {
	out := tgbotapi.NewMessage(msg.Chat.ID, text)
	out.ReplyToMessageID = msg.MessageID
	if _, err := b.api.Send(out); err != nil {
		log.Printf("send: %v", err)
	}
}

func (b *Bot) cmdNew(msg *tgbotapi.Message, text string, priority bool) {
	if text == "" {
		if priority {
			b.reply(msg, "Uso: /newp <texto>")
		} else {
			b.reply(msg, "Uso: /new <texto>")
		}
		return
	}
	id, err := b.store.Create(text, priority)
	if err != nil {
		log.Printf("create: %v", err)
		b.reply(msg, "Error al crear la tarea.")
		return
	}
	icon := "•"
	if priority {
		icon = "🔥"
	}
	b.reply(msg, fmt.Sprintf("%s Tarea #%d creada.", icon, id))
}

func (b *Bot) cmdList(msg *tgbotapi.Message) {
	tasks, err := b.store.ListOpen()
	if err != nil {
		log.Printf("list: %v", err)
		b.reply(msg, "Error al listar.")
		return
	}
	if len(tasks) == 0 {
		b.reply(msg, "Sin tareas abiertas. 🎉")
		return
	}
	var sb strings.Builder
	sb.WriteString("📌 Tareas abiertas:\n\n")
	for _, t := range tasks {
		if t.Priority {
			fmt.Fprintf(&sb, "🔥 #%d  %s\n", t.ID, t.Text)
		} else {
			fmt.Fprintf(&sb, "•  #%d  %s\n", t.ID, t.Text)
		}
	}
	b.reply(msg, sb.String())
}

// parseID splits "<id> <rest>" — returns the parsed id, the remaining text,
// and whether parsing succeeded.
func parseID(s string) (int64, string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", false
	}
	parts := strings.SplitN(s, " ", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}
	return id, rest, true
}

func (b *Bot) cmdEdit(msg *tgbotapi.Message, args string) {
	id, text, ok := parseID(args)
	if !ok || text == "" {
		b.reply(msg, "Uso: /edit <id> <texto>")
		return
	}
	n, err := b.store.UpdateText(id, text)
	if err != nil {
		log.Printf("edit: %v", err)
		b.reply(msg, "Error al editar.")
		return
	}
	if n == 0 {
		b.reply(msg, fmt.Sprintf("No existe la tarea #%d (o está cerrada).", id))
		return
	}
	b.reply(msg, fmt.Sprintf("Tarea #%d actualizada.", id))
}

func (b *Bot) cmdPriority(msg *tgbotapi.Message, args string) {
	id, _, ok := parseID(args)
	if !ok {
		b.reply(msg, "Uso: /priority <id>")
		return
	}
	now, err := b.store.TogglePriority(id)
	if err != nil {
		log.Printf("priority: %v", err)
		b.reply(msg, "Error al cambiar prioridad.")
		return
	}
	if now == nil {
		b.reply(msg, fmt.Sprintf("No existe la tarea #%d (o está cerrada).", id))
		return
	}
	if *now {
		b.reply(msg, fmt.Sprintf("🔥 Tarea #%d marcada como prioritaria.", id))
	} else {
		b.reply(msg, fmt.Sprintf("Tarea #%d sin prioridad.", id))
	}
}

func (b *Bot) cmdDone(msg *tgbotapi.Message, args string) {
	id, _, ok := parseID(args)
	if !ok {
		b.reply(msg, "Uso: /done <id>")
		return
	}
	n, err := b.store.MarkDone(id)
	if err != nil {
		log.Printf("done: %v", err)
		b.reply(msg, "Error al cerrar.")
		return
	}
	if n == 0 {
		b.reply(msg, fmt.Sprintf("No existe la tarea #%d (o ya estaba cerrada).", id))
		return
	}
	b.reply(msg, fmt.Sprintf("✅ Tarea #%d cerrada.", id))
}

func (b *Bot) cmdDelete(msg *tgbotapi.Message, args string) {
	id, _, ok := parseID(args)
	if !ok {
		b.reply(msg, "Uso: /delete <id>")
		return
	}
	n, err := b.store.Delete(id)
	if err != nil {
		log.Printf("delete: %v", err)
		b.reply(msg, "Error al borrar.")
		return
	}
	if n == 0 {
		b.reply(msg, fmt.Sprintf("No existe la tarea #%d.", id))
		return
	}
	b.reply(msg, fmt.Sprintf("🗑️ Tarea #%d borrada.", id))
}
