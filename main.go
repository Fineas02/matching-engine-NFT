package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	orderbook "github.com/fineas02/matching-engine/orderbook"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	ex := NewExchange()

	e.GET("/book/:market", ex.handleGetMarket)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	e.Start(":3000")
}

type OrderType string

const (
	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"
)

const (
	MarketETH orderbook.Market = "ETH"
)

type Exchange struct {
	orderbooks         map[orderbook.Market]*orderbook.Orderbook
	idToOrderMap       map[int64]*orderbook.Order
	orderIdToMarketMap map[int64]orderbook.Market
}

func NewExchange() *Exchange {
	orderbooks := make(map[orderbook.Market]*orderbook.Orderbook)
	idToOrderMap := make(map[int64]*orderbook.Order)
	orderIdToMarketMap := make(map[int64]orderbook.Market)
	orderbooks[MarketETH] = orderbook.NewOrderBook()

	return &Exchange{
		orderbooks:         orderbooks,
		idToOrderMap:       idToOrderMap,
		orderIdToMarketMap: orderIdToMarketMap,
	}
}

type PlaceOrderRequest struct {
	Type   OrderType
	Bid    bool
	Size   float64
	Price  float64
	Market orderbook.Market
}

type Order struct {
	ID        int64
	Price     float64
	Size      float64
	Bid       bool
	Timestamp int64
}

type OrderbookData struct {
	TotalBidVolume float64
	TotalAskVolume float64
	Asks           []*Order
	Bids           []*Order
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid order ID")
	}

	order, exists := ex.idToOrderMap[id]
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, "Order not found")
	}

	delete(ex.idToOrderMap, id)

	market, exists := ex.orderIdToMarketMap[id]
	if exists {
		ob := ex.orderbooks[market]
		ob.CancelOrder(order, market)
		delete(ex.orderIdToMarketMap, id)
	}

	return c.JSON(http.StatusOK, "Order cancelled successfully")
}

func (ex *Exchange) handleGetMarket(c echo.Context) error {
	market := orderbook.Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalBidVolume: ob.BidTotalVolume,
		TotalAskVolume: ob.AskTotalVolume,
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			o := Order{
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Asks = append(orderbookData.Asks, &o)
		}
	}

	for _, limit := range ob.Bids() {
		for _, order := range limit.Orders {
			o := Order{
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Bids = append(orderbookData.Bids, &o)
		}
	}

	return c.JSON(http.StatusOK, orderbookData)

}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := orderbook.Market(placeOrderData.Market)
	ob := ex.orderbooks[market]
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size)
	ex.idToOrderMap[order.ID] = order
	ex.orderIdToMarketMap[order.ID] = market

	if placeOrderData.Type == LimitOrder {
		ob.PlaceLimitOrder(placeOrderData.Price, order)
		return c.JSON(200, map[string]any{"msg": "limit order placed"})
	}
	if placeOrderData.Type == MarketOrder {
		matches := ob.PlaceMarketOrder(order)

		return c.JSON(200, map[string]any{"matches": len(matches)})
	}
	return nil
}
