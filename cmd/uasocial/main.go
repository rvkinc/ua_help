package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/rvkinc/uasocial/internal/service"
	"github.com/rvkinc/uasocial/internal/storage"

	"github.com/rvkinc/uasocial/config"
	"github.com/rvkinc/uasocial/internal/bot"
	"go.uber.org/zap"
)

//go:embed config.yml
var configfile []byte

func main() {
	var ctx, cancel = context.WithCancel(context.Background())
	var wg sync.WaitGroup
	defer wg.Wait()

	go func() { // listen for interrupt signal
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt)
		<-exit
		cancel()
	}()

	lg, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalln("new zap:", err)
	}

	cfg, err := config.NewConfig(configfile)
	if err != nil {
		log.Fatalln("parse config:", err)
	}

	st, err := storage.NewPostgres(cfg.StorageConfig)
	if err != nil {
		log.Fatalln("new postgres:", err)
	}

	b, err := bot.New(ctx, cfg.BotConfig, lg, service.NewService(st))
	if err != nil {
		log.Fatalln("run bot", err)
	}

	err = b.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
