package main

import (
	"math/rand"
	"time"

	"github.com/fineas02/matching-engine/client"
	mm "github.com/fineas02/matching-engine/marketmaker"
	"github.com/fineas02/matching-engine/server"
)

func main() {
	go server.StartServer()
	time.Sleep(1 * time.Second)

	c := client.NewClient()

	cfg := mm.Config{
		UserID:         0,
		OrderSize:      10,
		MinSpread:      20,
		MakeInterval:   1 * time.Second,
		SeedOffset:     40,
		ExchangeClient: c,
	}
	maker := mm.NewMarketMaker(cfg)

	maker.Start()

	time.Sleep(1 * time.Second)
	go marketOrderPlacer(c)

	select {}
}

func marketOrderPlacer(c *client.Client) {

	ticker := time.NewTicker(50 * time.Millisecond)

	for {
		randint := rand.Intn(10)
		bid := true
		if randint < 7 {
			bid = false
		}

		order := client.PlaceOrderParams{
			UserID: 1,
			Bid:    bid,
			Size:   1,
		}

		_, err := c.PlaceMarketOrder(&order)
		if err != nil {
			panic(err)
		}

		<-ticker.C
	}
}
