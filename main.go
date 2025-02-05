package main

import (
	"flag"
	"log"

	"net/http"
	_ "net/http/pprof"

	"github.com/chenx-dust/paracat/app"
	"github.com/chenx-dust/paracat/app/client"
	"github.com/chenx-dust/paracat/app/relay"
	"github.com/chenx-dust/paracat/app/server"
	"github.com/chenx-dust/paracat/config"
)

func main() {
	cfgFilename := flag.String("c", "config.json", "config file")
	enablePprof := flag.Bool("pprof", false, "enable pprof")
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Println("starting pprof server on localhost:6060")
			err := http.ListenAndServe("localhost:6060", nil)
			if err != nil {
				log.Fatalf("Failed to start pprof: %v", err)
			}
		}()
	}

	log.Println("loading config from", *cfgFilename)
	cfg, err := config.LoadFromFile(*cfgFilename)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// log.Println("config loaded:", cfg)

	var application app.App
	if cfg.Mode == config.ClientMode {
		application = client.NewClient(cfg)
	} else if cfg.Mode == config.ServerMode {
		application = server.NewServer(cfg)
	} else if cfg.Mode == config.RelayMode {
		application = relay.NewRelay(cfg)
	} else {
		log.Fatalf("Invalid mode: %v", cfg.Mode)
	}

	err = application.Run()
	if err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
