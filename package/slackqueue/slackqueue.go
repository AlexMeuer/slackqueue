package slackqueue

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/alexmeuer/slackqueue/internal/queue"
	"github.com/alexmeuer/slackqueue/internal/slackbot"
	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
)

func Run() {
	slackTkn, slackSecret := slackInfo()
	firestoreClient := firestoreClient()
	var store slackbot.QueueStore
	if firestoreClient != nil {
		log.Println("Using Google Cloud Datastore for queue storage.")
		store = queue.NewFirestoreStore(firestoreClient)
	} else {
		log.Println("Using in-memory queue storage.")
		store = queue.NewInMemoryStore()
	}

	bot, err := slackbot.New(slackTkn, store)
	if err != nil {
		log.Fatalln("Failed to start Slack bot:", err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGABRT, syscall.SIGTERM)

	r := mux.NewRouter()

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	r.HandleFunc("/slack", func(w http.ResponseWriter, r *http.Request) {
		verifier, err := slack.NewSecretsVerifier(r.Header, slackSecret)
		if err != nil {
			log.Println("Failed to create secrets verifier:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = verifier.Ensure()
		if err != nil {
			log.Println("Failed to verify secrets:", err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid request signature"))
			return
		}
		bot.HandleCommand(w, r)
	})

	srv := http.Server{
		Addr:              ":" + port(),
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

// slackInfo gets the slack token and signing secret from the environment.
// If either is not set, it will log a fatal error and exit.
func slackInfo() (string, string) {
	tkn, ok := os.LookupEnv("SLACK_TOKEN")
	if !ok {
		log.Fatal("SLACK_TOKEN not set")
	}
	secret, ok := os.LookupEnv("SLACK_SIGNING_SECRET")
	if !ok {
		log.Fatal("SLACK_SIGNING_SECRET not set")
	}
	return tkn, secret
}

func firestoreClient() *firestore.Client {
	projectID, ok := os.LookupEnv("GOOGLE_PROJECT_ID")
	if !ok || projectID == "" {
		return nil
	}
	client, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatalln("Failed to create Firestore client:", err)
	}
	return client
}

func port() string {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "4578"
	}
	return port
}
