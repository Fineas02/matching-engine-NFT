package main

import (
	"fmt"
	"time"

	"github.com/fineas02/matching-engine/client"
	mm "github.com/fineas02/matching-engine/marketmaker"
	"github.com/fineas02/matching-engine/server"
)

const ethPrice = 1962.0

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

	maker.Strart()

	time.Sleep(1 * time.Second)
	go marketOrderPlacer(c)

	select {}
}

func marketOrderPlacer(c *client.Client) {

	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		buyOrder := client.PlaceOrderParams{
			UserID: 1,
			Bid:    true,
			Size:   1,
		}

		_, err := c.PlaceMarketOrder(&buyOrder)
		if err != nil {
			panic(err)
		}

		sellOrder := client.PlaceOrderParams{
			UserID: 2,
			Bid:    false,
			Size:   1,
		}

		_, err = c.PlaceMarketOrder(&sellOrder)
		if err != nil {
			panic(err)
		}

		<-ticker.C
	}
}

func seedMarket(c *client.Client) {
	currentPrice := ethPrice //async call to fetch the price
	priceOffset := 100.0

	bidOrder := client.PlaceOrderParams{
		UserID: 0,
		Bid:    true,
		Price:  currentPrice - priceOffset,
		Size:   50,
	}
	_, err := c.PlaceLimitOrder(&bidOrder)
	if err != nil {
		panic(err)
	}
	askOrder := client.PlaceOrderParams{
		UserID: 0,
		Bid:    false,
		Price:  currentPrice + priceOffset,
		Size:   50,
	}
	_, err = c.PlaceLimitOrder(&askOrder)
	if err != nil {
		panic(err)
	}
}

func makeMarketSimple(c *client.Client) {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		bestAsk, err := c.GetBestAsk()
		if err != nil {
			panic(err)
		}
		bestBid, err := c.GetBestBid()
		if err != nil {
			panic(err)
		}

		if bestAsk == 0 && bestBid == 0 {
			seedMarket(c)
			continue
		}

		fmt.Println("best ask", bestAsk)
		fmt.Println("best bid", bestBid)

		<-ticker.C

	}
}
