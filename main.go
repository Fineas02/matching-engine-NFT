package main

import (
	"fmt"
	"time"

	"github.com/fineas02/matching-engine/client"
	"github.com/fineas02/matching-engine/server"
)

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	bidParams := &client.PlaceLimitOrderParams{
		UserID: 8,
		Bid:    true,
		Price:  10_000,
		Size:   10_0000,
	}

	go func() {
		for {
			resp, err := c.PlaceLimitOrder(bidParams)
			if err != nil {
				panic(err)
			}
			fmt.Println("order id => ", resp.OrderID)

			if err := c.CancelOrder(resp.OrderID); err != nil {
				panic(err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	askParams := &client.PlaceLimitOrderParams{
		UserID: 6,
		Bid:    false,
		Price:  7_000,
		Size:   10_0000,
	}

	for {

		resp, err := c.PlaceLimitOrder(askParams)
		if err != nil {
			panic(err)
		}
		fmt.Println("order id => ", resp.OrderID)

		time.Sleep(1 * time.Second)
	}

	select {}
}
