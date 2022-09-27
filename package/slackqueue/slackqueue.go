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
	"github.com/alexmeuer/slackqueue/internal/token"
	"github.com/gorilla/mux"
)

func Run() {
	clientID, clientSecret, _, devTkn := slackInfo()
	firestoreClient := firestoreClient()
	oauthFinalUrl := oauthFinalUrl()

	var queueStore slackbot.QueueStore
	var tknStore slackbot.TokenStore
	if firestoreClient != nil {
		log.Println("Using Google Cloud Datastore for queue and token storage.")
		queueStore = queue.NewFirestoreStore(firestoreClient)
		tknStore = token.NewFirestoreStore(firestoreClient)
	} else {
		log.Println("Using in-memory queue and token storage.")
		queueStore = queue.NewInMemoryStore()
		tknStore = &token.InMemoryStore{Token: devTkn}
	}

	bot, err := slackbot.New(context.Background(), clientID, clientSecret, tknStore, queueStore)
	if err != nil {
		log.Fatalln("Failed to start Slack bot:", err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGABRT, syscall.SIGTERM)

	r := mux.NewRouter()
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	r.HandleFunc("/oauth", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			log.Println("No code provided.")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("No code provided."))
			return
		}
		err := bot.ExhangeCodeForToken(r.Context(), code)
		if err != nil {
			log.Println("Failed to exchange code for token:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to exchange code for token."))
			return
		}
		w.Header().Set("Location", oauthFinalUrl)
		w.WriteHeader(http.StatusFound)
	})
	r.HandleFunc("/slack", func(w http.ResponseWriter, r *http.Request) {
		// verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		// if err != nil {
		// 	log.Println("Failed to create secrets verifier:", err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }
		// err = verifier.Ensure()
		// if err != nil {
		// 	log.Println("Failed to verify secrets:", err)
		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	w.Write([]byte("Invalid request signature"))
		// 	return
		// }
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

func slackInfo() (string, string, string, string) {
	clientID, clientIDOk := os.LookupEnv("SLACK_CLIENT_ID")
	clientSecret, clientSecretOk := os.LookupEnv("SLACK_CLIENT_SECRET")
	token, tokenOk := os.LookupEnv("SLACK_TOKEN")
	if !(clientIDOk && clientSecretOk) && !tokenOk {
		log.Fatalln("Either SLACK_TOKEN or both SLACK_CLIENT_ID and SLACK_CLIENT_SECRET must be set.")
	}
	signingSecret, signingSecretOk := os.LookupEnv("SLACK_SIGNING_SECRET")
	if !signingSecretOk {
		log.Fatalln("SLACK_SIGNING_SECRET must be set.")
	}
	return clientID, clientSecret, signingSecret, token
}

// oauthFinalUrl gets the final URL to redirect to after OAuth is complete from environment variables.
func oauthFinalUrl() string {
	url, ok := os.LookupEnv("OAUTH_FINAL_URL")
	if !ok {
		log.Fatalln("OAUTH_FINAL_URL must be set.")
	}
	return url
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
