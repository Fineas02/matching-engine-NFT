package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/fineas02/matching-engine/client"
	"github.com/fineas02/matching-engine/server"
)

const (
	maxOrders = 3
)

var (
	tick   = 1 * time.Second
	myAsks = make(map[float64]int64)
	myBids = make(map[float64]int64)
)

func marketOrderPlacer(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		marketSell := &client.PlaceOrderParams{
			UserID: 0,
			Bid:    false,
			Size:   1000,
		}
		orderResp, err := c.PlaceMarketOrder(marketSell)
		if err != nil {
			log.Println(orderResp.OrderID)
		}
		marketBuyOrder := &client.PlaceOrderParams{
			UserID: 0,
			Bid:    true,
			Size:   1000,
		}
		orderResp, err = c.PlaceMarketOrder(marketBuyOrder)
		if err != nil {
			log.Println(orderResp.OrderID)
		}
		<-ticker.C
	}
}

func makeMarketSimple(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		bestAsk, err := c.GetBestAsk()
		if err != nil {
			log.Println(err)
		}
		bestBid, err := c.GetBestBid()
		if err != nil {
			log.Println(err)
		}

		spread := math.Abs(bestBid - bestAsk)
		fmt.Println("exchange spread ", spread)

		if len(myBids) < 3 {
			bidLimit := &client.PlaceOrderParams{
				UserID: 1,
				Bid:    true,
				Price:  bestBid + 100,
				Size:   1000,
			}
			bidOrderResp, err := c.PlaceLimitOrder(bidLimit)
			if err != nil {
				log.Println(bidOrderResp.OrderID)
			}
			myBids[bidLimit.Price] = bidOrderResp.OrderID
		}
		if len(myAsks) < 3 {
			askLimit := &client.PlaceOrderParams{
				UserID: 1,
				Bid:    false,
				Price:  bestAsk - 100,
				Size:   1000,
			}
			askOrderResp, err := c.PlaceLimitOrder(askLimit)
			if err != nil {
				log.Println(askOrderResp.OrderID)
			}
			myAsks[askLimit.Price] = askOrderResp.OrderID
		}

		fmt.Println("best ask price ", bestAsk)
		fmt.Println("best bid price ", bestBid)
		<-ticker.C
	}
}

func seedMarket(c *client.Client) error {
	ask := &client.PlaceOrderParams{
		UserID: 1,
		Bid:    false,
		Price:  10_000,
		Size:   1_000_000,
	}
	bid := &client.PlaceOrderParams{
		UserID: 0,
		Bid:    true,
		Price:  9_000,
		Size:   1_000_000,
	}
	ask2 := &client.PlaceOrderParams{
		UserID: 0,
		Bid:    false,
		Price:  9_000,
		Size:   1_000_000,
	}
	market := &client.PlaceOrderParams{
		UserID: 2,
		Bid:    true,
		Size:   1_500_000,
	}
	_, err := c.PlaceLimitOrder(ask)
	if err != nil {
		return err
	}
	_, err = c.PlaceLimitOrder(ask2)
	if err != nil {
		return err
	}
	_, err = c.PlaceLimitOrder(bid)
	if err != nil {
		return err
	}
	_, err = c.PlaceMarketOrder(market)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	if err := seedMarket(c); err != nil {
		panic(err)
	}

	go makeMarketSimple(c)

	time.Sleep(1 * time.Second)
	marketOrderPlacer(c)
	// limitOrderParams := &client.PlaceOrderParams{
	// 	UserID: 1,
	// 	Bid:    false,
	// 	Price:  10_000,
	// 	Size:   5_000_000,
	// }
	// _, err := c.PlaceLimitOrder(limitOrderParams)
	// if err != nil {
	// 	panic(err)
	// }
	// // fmt.Println("placed limit order from the client -> ", resp.OrderID)
	// otherLimitOrderParams := &client.PlaceOrderParams{
	// 	UserID: 2,
	// 	Bid:    true,
	// 	Price:  8_000,
	// 	Size:   1_000_000,
	// }
	// _, err = c.PlaceLimitOrder(otherLimitOrderParams)
	// if err != nil {
	// 	panic(err)
	// }
	// // fmt.Println("placed limit order from the client -> ", resp.OrderID)
	// askLimitOrderParams := &client.PlaceOrderParams{
	// 	UserID: 2,
	// 	Bid:    true,
	// 	Price:  10_000,
	// 	Size:   3_000_000,
	// }
	// _, err = c.PlaceLimitOrder(askLimitOrderParams)
	// if err != nil {
	// 	panic(err)
	// }

	// marketOrderParams := &client.PlaceOrderParams{
	// 	UserID: 0,
	// 	Bid:    false,
	// 	Size:   2_000_000,
	// }
	// _, err = c.PlaceMarketOrder(marketOrderParams)
	// if err != nil {
	// 	panic(err)
	// }

	// bestBidPrice, err := c.GetBestBid()
	// if err != nil {
	// 	panic(err)
	// }
	// bestAskPrice, err := c.GetBestAsk()
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("best bid price ", bestBidPrice)
	// fmt.Println("best ask price ", bestAskPrice)
	// // fmt.Println("placed market order from the client -> ", resp.OrderID)

	// time.Sleep(1 * time.Second)

	select {}
}
