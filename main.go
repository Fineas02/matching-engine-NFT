package main

import (
	"time"

	"github.com/fineas02/matching-engine/client"
	"github.com/fineas02/matching-engine/server"
)

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	for {
		limitOrderParams := &client.PlaceOrderParams{
			UserID: 1,
			Bid:    false,
			Price:  10_000,
			Size:   1_000_000,
		}
		_, err := c.PlaceLimitOrder(limitOrderParams)
		if err != nil {
			panic(err)
		}
		// fmt.Println("placed limit order from the client -> ", resp.OrderID)
		otherLimitOrderParams := &client.PlaceOrderParams{
			UserID: 2,
			Bid:    false,
			Price:  9_000,
			Size:   1_000_000,
		}
		_, err = c.PlaceLimitOrder(otherLimitOrderParams)
		if err != nil {
			panic(err)
		}
		// fmt.Println("placed limit order from the client -> ", resp.OrderID)

		marketOrderParams := &client.PlaceOrderParams{
			UserID: 0,
			Bid:    true,
			Size:   2_000_000,
		}
		_, err = c.PlaceMarketOrder(marketOrderParams)
		if err != nil {
			panic(err)
		}

		// fmt.Println("placed market order from the client -> ", resp.OrderID)

		time.Sleep(1 * time.Second)
	}
	select {}
}
