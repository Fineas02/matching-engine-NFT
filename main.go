package main

import (
	"log"
	"time"

	"github.com/fineas02/matching-engine/client"
	"github.com/fineas02/matching-engine/server"
)

func main() {
	exchange, err := server.NewExchange()
	if err != nil {
		log.Fatalf("Failed to create Exchange: %v", err)
	}

	go server.StartServer(exchange)
	time.Sleep(1 * time.Second)

	c := client.NewClient()

	// cfg := mm.Config{
	// 	UserID:         0,
	// 	Leverage:       10,
	// 	OrderSize:      10,
	// 	MinSpread:      20,
	// 	MakeInterval:   3 * time.Second,
	// 	SeedOffset:     40,
	// 	ExchangeClient: c,
	// }
	// maker := mm.NewMarketMaker(cfg)

	// maker.Start()

	// time.Sleep(1 * time.Second)
	// go marketOrderPlacer(c)
	// Placing limit orders for user 0
	_, err = c.PlaceLimitOrder(&client.PlaceOrderParams{
		UserID:   0,
		Bid:      true,
		Size:     10,
		Price:    990,
		Leverage: 10,
	})
	if err != nil {
		panic(err)
	}

	_, err = c.PlaceLimitOrder(&client.PlaceOrderParams{
		UserID:   0,
		Bid:      false,
		Size:     10,
		Price:    1010,
		Leverage: 1,
	})
	if err != nil {
		panic(err)
	}

	// Sleep to ensure limit orders are placed
	time.Sleep(2 * time.Second)

	// Placing market orders for user 1 to fill limit orders
	_, err = c.PlaceMarketOrder(&client.PlaceOrderParams{
		UserID:   1,
		Bid:      true, // This should fill user 0's ask order
		Size:     10,
		Leverage: 1,
	})
	if err != nil {
		panic(err)
	}

	_, err = c.PlaceMarketOrder(&client.PlaceOrderParams{
		UserID:   1,
		Bid:      false, // This should fill user 0's bid order
		Size:     10,
		Leverage: 5,
	})
	if err != nil {
		panic(err)
	}

	// select {}
}

// func marketOrderPlacer(c *client.Client) {

// 	ticker := time.NewTicker(5000 * time.Millisecond)

// 	for {
// 		randint := rand.Intn(10)
// 		bid := true
// 		if randint < 7 {
// 			bid = false
// 		}

// 		order := client.PlaceOrderParams{
// 			UserID:   1,
// 			Bid:      bid,
// 			Size:     1,
// 			Leverage: 10,
// 		}

// 		_, err := c.PlaceMarketOrder(&order)
// 		if err != nil {
// 			panic(err)
// 		}

// 		<-ticker.C
// 	}
// }
