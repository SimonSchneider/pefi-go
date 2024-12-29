package pefi

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/SimonSchneider/goslu/config"
	"github.com/SimonSchneider/goslu/migrate"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"github.com/SimonSchneider/pefigo"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getEnv func(string) string, getwd func() (string, error)) error {
	cfg, err := parseConfig(args[1:], getEnv)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()
	logger := srvu.LogToOutput(log.New(stdout, "", log.LstdFlags|log.Lshortfile))

	db, err := GetMigratedDB(ctx, pefigo.StaticEmbeddedFS, "static/migrations", cfg.DbURL)
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	public, tmpls, err := templ.GetPublicAndTemplates(pefigo.StaticEmbeddedFS, &templ.Config{
		Watch:        cfg.Watch,
		TmplPatterns: []string{"templates/*.gohtml"},
		RootTmplProvider: func() *template.Template {
			return template.New("").Funcs(template.FuncMap{
				"mkSlice": func(args ...interface{}) []interface{} {
					return args
				},
				"dict": func(args ...interface{}) map[string]interface{} {
					if len(args)%2 != 0 {
						panic("illegal number of args to dict function")
					}
					m := make(map[string]interface{}, len(args)/2)
					for i := 0; i < len(args); i += 2 {
						if key, ok := args[i].(string); !ok {
							panic("illegal key arg to dict function")
						} else {
							m[key] = args[i+1]
						}
					}
					return m
				},
			})
		},
	})
	if err != nil {
		return fmt.Errorf("sub static: %w", err)
	}

	srv := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Addr:    cfg.Addr,
		Handler: srvu.With(NewHandler(db, public, tmpls), srvu.WithCompression(), srvu.WithLogger(logger)),
	}
	logger.Printf("starting chore server, listening on %s\n  sqliteDB: %s", cfg.Addr, cfg.DbURL)
	return srvu.RunServerGracefully(ctx, srv, logger)
}

type Config struct {
	Addr  string
	Watch bool
	DbURL string
}

func parseConfig(args []string, getEnv func(string) string) (cfg Config, err error) {
	err = config.ParseInto(&cfg, flag.NewFlagSet("", flag.ExitOnError), args, getEnv)
	return cfg, err
}

func GetMigratedDB(ctx context.Context, dir fs.FS, path string, conn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	migrations, err := fs.Sub(dir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations: %w", err)
	}
	if err := migrate.Migrate(ctx, migrations, db); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}
	return db, nil
}
