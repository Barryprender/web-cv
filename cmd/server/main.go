// Command server runs barrypre.com: Barry Prendergast's CV site.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"barrypre.com/webcv/internal/data"
	"barrypre.com/webcv/internal/mail"
	"barrypre.com/webcv/internal/site"
)

func main() {
	addr := ":" + envOr("PORT", "8080")

	// Refusing to start beats starting with a contact form that cannot
	// deliver: the failure is a deployment mistake, and it should surface here
	// rather than as the first lost message.
	sender, err := mail.FromEnv(log.Printf, data.Me.Contact.Email)
	if err != nil {
		log.Fatalf("contact email: %v", err)
	}

	handler := site.NewHandler(time.Now().Year(), sender)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
