package slackqueue

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexmeuer/slackqueue/internal/queue"
	"github.com/alexmeuer/slackqueue/internal/slackbot"
	"github.com/gorilla/mux"
)

func Run() {
	slackTkn, ok := os.LookupEnv("SLACK_TOKEN")
	if !ok {
		log.Fatal("SLACK_TOKEN not set")
	}
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "4578"
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGABRT, syscall.SIGTERM)

	r := mux.NewRouter()

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	log.Println("Starting Slack bot.")
	slack, err := slackbot.New(slackTkn, queue.NewInMemoryStore())
	if err != nil {
		log.Fatalln("Failed to start Slack bot:", err)
	}
	r.HandleFunc("/slack", func(w http.ResponseWriter, r *http.Request) {
		// TODO: verify the request came from Slack via signing secret.
		slack.HandleCommand(w, r)
	})

	srv := http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 0,
		WriteTimeout:      15 * time.Second,
	}

	go func() {
		defer close(sig)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
	log.Println("Listening for HTTP on", srv.Addr)

	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
