package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"github.com/SimonSchneider/pefigo"
	"github.com/SimonSchneider/pefigo/internal/service"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	cfg, err := service.ParseConfig(args[1:], getEnv)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	logger := srvu.LogToOutput(log.New(stdout, "", log.LstdFlags|log.Lshortfile))

	db, err := service.GetMigratedDB(ctx, pefigo.StaticEmbeddedFS, "static/migrations", cfg.DbURL)
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	svc := service.New(db)

	public, _, err := templ.GetPublicAndTemplates(pefigo.StaticEmbeddedFS, &templ.Config{
		Watch: cfg.Watch,
	})
	if err != nil {
		return fmt.Errorf("sub static: %w", err)
	}

	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.Addr,
		Handler: srvu.With(NewHandler(svc, public), http.NewCrossOriginProtection().Handler, srvu.WithCompression(), srvu.WithLogger(logger)),
	}
	logger.Printf("starting chore server, listening on %s\n  sqliteDB: %s", cfg.Addr, cfg.DbURL)
	return srvu.RunServerGracefully(ctx, srv, logger)
}
