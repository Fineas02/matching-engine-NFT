package orderbook

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimit(t *testing.T) {
	l := NewLimit(10_000)
	buyOrderA := NewOrder(true, 5)
	buyOrderB := NewOrder(true, 8)
	buyOrderC := NewOrder(true, 10)

	l.AddOrder(buyOrderA)
	l.AddOrder(buyOrderB)
	l.AddOrder(buyOrderC)

	l.DeleteOrder(buyOrderB)

	fmt.Println(l)
}

func TestPlaceLimitOrder(t *testing.T) {
	ob := NewOrderBook()

	sellOrderA := NewOrder(false, 10)
	sellOrderB := NewOrder(false, 100)
	ob.PlaceLimitOrder(10_000, sellOrderA)
	ob.PlaceLimitOrder(40_000, sellOrderB)

	assert(t, len(ob.asks), 2)
}

func TestPlaceMarketOrder(t *testing.T) {
	ob := NewOrderBook()

	sellOrder := NewOrder(false, 20)
	ob.PlaceLimitOrder(10_000, sellOrder)

	buyOrder := NewOrder(true, 10)
	matches := ob.PlaceMarketOrder(buyOrder)

	assert(t, len(matches), 1)
	assert(t, len(ob.asks), 1)
	assert(t, ob.AskTotalVolume, 10.0)
	assert(t, matches[0].Ask, sellOrder)
	assert(t, matches[0].Bid, buyOrder)
	assert(t, buyOrder.IsFilled(), true)

	fmt.Printf("%+v", matches)

}

func TestPlaceMarketOrderMultiFill(t *testing.T) {
	ob := NewOrderBook()

	buyOrderA := NewOrder(true, 5)
	buyOrderB := NewOrder(true, 8)
	buyOrderC := NewOrder(true, 10)
	buyOrderD := NewOrder(true, 1)

	ob.PlaceLimitOrder(10_000, buyOrderA)
	ob.PlaceLimitOrder(9_000, buyOrderB)
	ob.PlaceLimitOrder(5_000, buyOrderC)
	ob.PlaceLimitOrder(5_000, buyOrderD)

	assert(t, ob.BidTotalVolume, 24.0)

	sellOrder := NewOrder(false, 20)
	matches := ob.PlaceMarketOrder(sellOrder)

	assert(t, ob.BidTotalVolume, 4.0)
	assert(t, len(matches), 3)
	assert(t, len(ob.bids), 1)
}

func TestPlaceLimitOrderPartialFill(t *testing.T) {
    ob := NewOrderBook()

    // place a sell limit order
    sellOrder := NewOrder(false, 20)
    ob.PlaceLimitOrder(10000, sellOrder)

    assert(t, len(ob.asks), 1)
    assert(t, ob.AskTotalVolume, 20.0)

    // place a buy market order that is smaller than the sell limit order
    buyOrder := NewOrder(true, 10)
    matches := ob.PlaceMarketOrder(buyOrder)

    assert(t, len(matches), 1) // should be one match
    assert(t, len(ob.asks), 1) // should still be one sell limit order
    assert(t, ob.AskTotalVolume, 10.0) // volume of sell limit order should have decreased
    assert(t, matches[0].Ask, sellOrder) // sell order in match should be the sell limit order
    assert(t, matches[0].Bid, buyOrder) // buy order in match should be the market order
    assert(t, matches[0].SizeFilled, 10.0) // size filled should be size of market order
    assert(t, buyOrder.IsFilled(), true) // market order should be filled
    assert(t, sellOrder.IsFilled(), false) // sell limit order should not be filled
}

// func TestCancelOrder(t *testing.T) {
// 	ob := NewOrderBook()
// 	buyOrder := NewOrder(true, 4)
// 	ob.PlaceLimitOrder(10_000, buyOrder)

// 	assert(t, ob.BidTotalVolume(), 4.0)

// 	ob.CancelOrder(buyOrder)

// 	assert(t, ob.BidTotalVolume(), 0.0)
// }
