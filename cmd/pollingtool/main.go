package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/qolors/gosrs/internal/infra/api/osrsclient"
	"github.com/qolors/gosrs/internal/infra/notifier"
	"github.com/qolors/gosrs/internal/infra/storage"
	"github.com/qolors/gosrs/internal/services/courier"
	"github.com/qolors/gosrs/internal/services/poller"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading Discord Webhook: %v", err)
		return
	}

	wh := os.Getenv("DISCORD_WEBHOOK")

	client := osrsclient.NewOSRSClient("Wooooo91")
	storage := storage.NewRingBuffer(1440)
	notifier := notifier.NewDiscordNotifier(wh)
	courier := courier.NewCourier(notifier)

	poller := poller.NewPoller(client, storage, courier)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			if err := poller.Poll(); err != nil {
				log.Printf("Poll error: %v\n", err)
			}
		}
	}()

	select {}

}
