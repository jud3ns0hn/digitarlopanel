// Command digitarlopanel is the entry point for the DigitarloPanel server.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/api"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/config"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/database"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/certgen"
	"github.com/jud3ns0hn/digitarlopanel/backend/web"
)

func main() {
	configPath := flag.String("config", "/etc/digitarlopanel/config.json", "path to config file")
	listen := flag.String("listen", "", "override listen address (e.g. :8088)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *listen != "" {
		cfg.Listen = *listen
	}

	db, adminPassword, err := database.Init(cfg.DBPath())
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	if adminPassword != "" {
		fmt.Fprintln(os.Stderr, "============================================================")
		fmt.Fprintln(os.Stderr, " DigitarloPanel first run - default admin account created")
		fmt.Fprintln(os.Stderr, "   Username: admin")
		fmt.Fprintf(os.Stderr, "   Password: %s\n", adminPassword)
		fmt.Fprintln(os.Stderr, " Change this password after your first login.")
		fmt.Fprintln(os.Stderr, "============================================================")
	}

	srv := api.NewServer(cfg, db, *configPath)
	handler := srv.Handler(web.FS())

	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if cfg.TLSEnabled {
		certPath := cfg.TLSCert
		keyPath := cfg.TLSKey
		if certPath == "" {
			certPath = cfg.DataDir + "/panel-cert.pem"
		}
		if keyPath == "" {
			keyPath = cfg.DataDir + "/panel-key.pem"
		}
		if cfg.TLSAutoSelfSigned {
			if err := certgen.EnsureSelfSigned(certPath, keyPath); err != nil {
				log.Fatalf("generate self-signed certificate: %v", err)
			}
		}
		log.Printf("DigitarloPanel listening on https://%s", cfg.Listen)
		if err := httpServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	log.Printf("DigitarloPanel listening on http://%s", cfg.Listen)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
