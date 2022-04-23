package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

type GGDialogsResponce struct {
	List []struct {
		ID   int `json:"id"`
		User struct {
			ID       int    `json:"id"`
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
			ObjKey   string `json:"obj_key"`
		} `json:"user"`
		Unread      int    `json:"unread"`
		Deleted     int    `json:"deleted"`
		LastMessage string `json:"last_message"`
		LastAuthor  int    `json:"last_author"`
	} `json:"list"`
	Total  string `json:"total"`
	Unread int    `json:"unread"`
}

func Run(f func(ctx context.Context, log *zap.Logger) error) {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	// No graceful shutdown.
	ctx := context.Background()
	if err := f(ctx, log); err != nil {
		log.Fatal("Run failed", zap.Error(err))
	}
	// Done.
}

func getResponce(log *zap.Logger) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://goodgame.ru/ajax/dialogs/get", nil)
	if err != nil {
		// handle error
	}
	phpsessid := os.Getenv("PHPSESSID")
	req.Header.Add("Referer", "https://goodgame.ru/")
	req.Header.Add("Cookie", "PHPSESSID="+phpsessid+";")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Client failed", zap.Error(err))
		return "Client failed"
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Responce failed", zap.Error(err))
		return "Responce failed"
	}
	var ggResp GGDialogsResponce
	err = json.Unmarshal(body, &ggResp)
	if err != nil {
		fmt.Println("error:", err)
	}

	// log.Info("Got body resp: " + string(body))
	// log.Info("Got responce: ", zap.Any(resp))
	return string(ggResp.Total)
}

func main() {
	// Environment variables:
	//	BOT_TOKEN:     token from BotFather
	// 	APP_ID:        app_id of Telegram app.
	// 	APP_HASH:      app_hash of Telegram app.
	// 	SESSION_FILE:  path to session file
	// 	SESSION_DIR:   path to session directory, if SESSION_FILE is not Setting
	Run(func(ctx context.Context, log *zap.Logger) error {
		// Dispatcher handles incoming updates.
		dispatcher := tg.NewUpdateDispatcher()
		opts := telegram.Options{
			Logger:        log,
			UpdateHandler: dispatcher,
		}
		return telegram.BotFromEnvironment(ctx, opts, func(ctx context.Context, client *telegram.Client) error {
			// Raw MTProto API client, allows making raw RPC calls.
			api := tg.NewClient(client)

			// Helper for sending messages.
			sender := message.NewSender(api)

			// Setting up handler for incoming message.
			dispatcher.OnNewMessage(func(ctx context.Context, entities tg.Entities, u *tg.UpdateNewMessage) error {
				m, ok := u.Message.(*tg.Message)
				if !ok || m.Out {
					// Outgoing message, not interesting.
					log.Fatal("Something went wrong")
					return nil

				}

				// Sending reply.

				_, err := sender.Reply(entities, u).Text(ctx, getResponce(log))
				return err
			})
			return nil
		}, telegram.RunUntilCanceled)
	})
}
