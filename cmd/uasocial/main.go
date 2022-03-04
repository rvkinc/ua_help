package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/google/uuid"
	"github.com/rvkinc/uasocial/config"
	"github.com/rvkinc/uasocial/internal/storage"
	"go.uber.org/zap"
	"log"
)

//go:embed config.yml
var configfile []byte

func main() {

	cfg, err := config.NewConfig(configfile)
	if err != nil {
		log.Fatal("parse config", zap.Error(err))
	}

	p, err := storage.NewPostgres(cfg.StorageConfig)
	if err != nil {
		log.Fatalln("new postgres:", err)
	}

	ctx := context.Background()
	u, err := p.UpsertUser(ctx, &storage.User{
		TgID:     1902842,
		ChatID:   4738948,
		Name:     "MyName1",
		Language: "UA",
	})

	if err != nil {
		log.Fatalln("upsert user", err)
	}

	fmt.Println(*u)

	r, err := p.SelectLocalityRegions(ctx, "Львів")
	if err != nil {
		log.Fatalln("select locality regions", err)
	}

	fmt.Println(r)

	catID, err := uuid.Parse("0fe99167-d809-4194-99d9-9c3d66664d1f")
	if err != nil {
		log.Fatalln("parse category id", err)
	}

	err = p.InsertHelp(ctx, &storage.HelpInsert{
		CreatorID:   uuid.New(),
		CategoryIDs: []uuid.UUID{catID},
		LocalityID:  23906,
		Description: "some text here",
	})

	if err != nil {
		log.Fatalln("insert help:", err)
	}

	v, err := p.SelectHelpsByLocalityCategory(ctx, 23906, catID)
	if err != nil {
		log.Fatalln("SelectHelpsByLocalityCategory:", err)
	}

	fmt.Println(v)

	uid, err := uuid.Parse("bce8fa02-210c-4606-bbba-8826bdaafe3e")
	if err != nil {
		log.Fatalln("parse uid:", err)
	}

	uh, err := p.SelectHelpsByUser(ctx, uid)
	if err != nil {
		log.Fatalln("SelectHelpsByLocalityCategory:", err)
	}

	fmt.Println(uh)

	// #############################################################################

	// var ctx, cancel = context.WithCancel(context.Background())
	// var wg sync.WaitGroup
	// defer wg.Wait()
	//
	// go func() { // listen for interrupt signal
	// 	exit := make(chan os.Signal, 1)
	// 	signal.Notify(exit, os.Interrupt)
	// 	<-exit
	// 	cancel()
	// }()
	//
	// lg, err := zap.NewDevelopment()
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	//
	// cfg, err := config.NewConfig(configfile)
	// if err != nil {
	// 	lg.Fatal("parse config", zap.Error(err))
	// }
	//
	// b, err := bot.New(ctx, cfg.BotConfig, lg, nil)
	// if err != nil {
	// 	lg.Fatal("run bot", zap.Error(err))
	// }
	//
	// err = b.Run()
	// if err != nil {
	// 	log.Fatalln(err)
	// }
}
