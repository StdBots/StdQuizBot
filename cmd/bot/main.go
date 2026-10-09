package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/StdBots/StdQuizBot/internal/bot"
	"github.com/StdBots/StdQuizBot/internal/config"
	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
)

const Banner = `
  ____ _____ ____     ___  _   _ ___ _____ ____   ___ _____ 
 / ___|_   _|  _ \   / _ \| | | |_ _|__  / __ ) / _ \_   _|
 \___ \ | | | | | | | | | | | | || |  / /|  _ \| | | || |  
  ___) || | | |_| | | |_| | |_| || | / /_| |_) | |_| || |  
 |____/ |_| |____/   \__\_\\___/|___/____|____/ \___/ |_|  
                                                            
   Enterprise Telegram Quiz & Multiplayer Tournament Engine
   Developer: STD DEEPANSHU (https://deepanshu.in) | @STDBOTS
`

func main() {
	fmt.Println(Banner)

	// Step 1: Security & Credit Integrity Verification
	ok, issues := credit.VerifyIntegrity()
	if !ok {
		log.Fatalf("[FATAL] Tamper detected in core credit modules: %v", issues)
	}

	// Step 2: Load runtime configuration
	cfg := config.Load()

	// Step 3: Connect to MongoDB
	db, err := database.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("[FATAL] Could not connect to MongoDB: %v", err)
	}
	defer db.Disconnect()

	// Step 4: Initialize bot instance
	botInstance, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("[FATAL] Bot initialization failed: %v", err)
	}

	// Step 5: Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n[SYSTEM] Received shutdown signal. Gracefully stopping StdQuizBot...")
		db.Disconnect()
		os.Exit(0)
	}()

	// Step 6: Start listening for messages and poll updates
	botInstance.Start()
}
