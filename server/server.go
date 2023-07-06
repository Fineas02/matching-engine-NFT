package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/fineas02/matching-engine/margin"
	orderbook "github.com/fineas02/matching-engine/orderbook"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const (
	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"
	MarketETH   Market    = "ETH"
)

type (
	OrderType string
	Market    string

	PlaceOrderRequest struct {
		UserID   int64
		Leverage float64
		Type     OrderType
		Bid      bool
		Size     float64
		Price    float64
		Market   Market
	}

	Order struct {
		UserID    int64
		ID        int64
		Price     float64
		Size      float64
		Bid       bool
		Timestamp int64
	}

	OrderbookData struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks           []*Order
		Bids           []*Order
	}

	MatchedOrder struct {
		UserID int64
		Price  float64
		Size   float64
		ID     int64
	}

	APIError struct {
		Error string
	}
)

func StartServer(ex *Exchange) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	ex.registerUser(0)
	ex.registerUser(1)
	ex.registerUser(2)

	e.GET("/trades/:market", ex.handleGetTrades)
	e.GET("/book/:market", ex.handleGetMarket)
	e.GET("/order/:userID", ex.handleGetOrders)
	e.POST("/order", ex.handlePlaceOrder)

	e.DELETE("/order/:id", ex.cancelOrder)

	e.GET("book/:market/bid", ex.handleGetBestBid)
	e.GET("book/:market/ask", ex.handleGetBestAsk)

	e.Start(":3000")
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

func NewMarketConfig(initialMarginRequirement, maximumLeverage, maintenanceMargin, tickSize, minOrder, quantityStep float64) *margin.MarketConfig {
	return &margin.MarketConfig{
		InitialMarginRequirement: initialMarginRequirement,
		MaximumLeverage:          maximumLeverage,
		MaintenanceMargin:        maintenanceMargin,
		TickSize:                 tickSize,
		MinOrder:                 minOrder,
		QuantityStep:             quantityStep,
	}
}

type Exchange struct {
	mu    sync.RWMutex
	Users map[int64]*margin.User

	MarketConfig map[Market]*margin.MarketConfig

	// Orders maps users to their orders
	Orders     map[int64][]*orderbook.Order
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchange() (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketETH] = orderbook.NewOrderbook()

	marketConfigs := make(map[Market]*margin.MarketConfig)
	marketConfigs[MarketETH] = NewMarketConfig(0.10, 10.0, 0.05, 0.01, 0.01, 0.001) // use marketConfigs

	return &Exchange{
		Users:        make(map[int64]*margin.User),
		Orders:       make(map[int64][]*orderbook.Order),
		orderbooks:   orderbooks,
		MarketConfig: marketConfigs,
	}, nil
}

func (ex *Exchange) registerUser(userID int64) {
	user := margin.NewUser(userID)
	ex.Users[user.ID] = user

	logrus.WithFields(logrus.Fields{
		"balance": user.Balance,
		"id":      userID,
	}).Info("new exchange user")
}

func (ex *Exchange) handleGetTrades(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, APIError{Error: "orderbook not found"})
	}

	return c.JSON(http.StatusOK, ob.Trades)
}

type PriceResponse struct {
	Price float64
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	var (
		market = Market(c.Param("market"))
		ob     = ex.orderbooks[market]
		order  = Order{}
	)

	if len(ob.Bids()) == 0 {
		return c.JSON(http.StatusOK, order)
	}

	bestLimit := ob.Bids()[0]
	bestOrder := bestLimit.Orders[0]

	order.Price = bestLimit.Price
	order.UserID = bestOrder.UserID

	return c.JSON(http.StatusOK, order)
}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {
	var (
		market = Market(c.Param("market"))
		ob     = ex.orderbooks[market]
		order  = Order{}
	)

	if len(ob.Asks()) == 0 {
		return c.JSON(http.StatusOK, order)
	}

	bestLimit := ob.Asks()[0]
	bestOrder := bestLimit.Orders[0]

	order.Price = bestLimit.Price
	order.UserID = bestOrder.UserID

	return c.JSON(http.StatusOK, order)
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	ob := ex.orderbooks[MarketETH]
	order := ob.Orders[int64(id)]

	ob.CancelOrder(order)

	log.Println("order canceled id => ", id)

	return c.JSON(200, map[string]any{"msg": "order deleted"})
}

type GetOrdersResponse struct {
	Asks []Order
	Bids []Order
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userIDStr := c.Param("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return err
	}

	ex.mu.RLock()
	orderbookOrders := ex.Orders[int64(userID)]
	ordersResp := &GetOrdersResponse{
		Asks: []Order{},
		Bids: []Order{},
	}

	for i := 0; i < len(orderbookOrders); i++ {
		// If the limit hasn't been cleared yet, the filled orders at that level
		// will be appended to the get orders. Check if limit is nil to avoid this
		if orderbookOrders[i].Limit == nil {
			continue
		}

		order := Order{
			ID:        orderbookOrders[i].ID,
			UserID:    orderbookOrders[i].UserID,
			Price:     orderbookOrders[i].Limit.Price,
			Size:      orderbookOrders[i].Size,
			Timestamp: orderbookOrders[i].Timestamp,
			Bid:       orderbookOrders[i].Bid,
		}

		if order.Bid {
			ordersResp.Bids = append(ordersResp.Bids, order)
		} else {
			ordersResp.Asks = append(ordersResp.Asks, order)
		}
	}

	ex.mu.RUnlock()
	return c.JSON(http.StatusOK, ordersResp)
}

func (ex *Exchange) handleGetMarket(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			o := Order{
				UserID:    order.UserID,
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
				UserID:    order.UserID,
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

// Check if the order size is within the maximum allowable size given the user's balance and the market's max leverage
func (ex *Exchange) handleCheckMaxContractSize(userID int64, market Market, orderSize float64) error {
	ex.mu.RLock()
	user, userExists := ex.Users[userID]
	ex.mu.RUnlock()

	if !userExists {
		return fmt.Errorf("user not found")
	}

	marketConfig, marketExists := ex.MarketConfig[market]

	if !marketExists {
		return fmt.Errorf("market not found")
	}

	equity := user.UpdateEquity()
	price := ex.calculatePrice(market) // assuming you have a method to get the current price
	maxContractSize := (equity * marketConfig.MaximumLeverage) / price

	if orderSize > maxContractSize {
		return fmt.Errorf("order size too large: balance %f, maxContractSize %f, order size %f",
			equity, maxContractSize, orderSize)
	}

	return nil
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}

	totalSizeFilled := 0.0
	sumPrice := 0.0
	for i := 0; i < len(matchedOrders); i++ {

		id := matches[i].Bid.ID
		limitUserID := matches[i].Bid.UserID
		if isBid {
			limitUserID = matches[i].Ask.UserID
			id = matches[i].Ask.ID
		}

		matchedOrders[i] = &MatchedOrder{
			UserID: limitUserID,
			ID:     id,
			Size:   matches[i].SizeFilled,
			Price:  matches[i].Price,
		}

		totalSizeFilled += matches[i].SizeFilled
		sumPrice += matches[i].Price
	}

	avgPrice := sumPrice / float64(len(matches))

	logrus.WithFields(logrus.Fields{
		"size":     totalSizeFilled,
		"type":     order.Type,
		"avgPrice": avgPrice,
		"orderID":  order.ID,
	}).Info("filled market order")

	newOrderMap := make(map[int64][]*orderbook.Order)
	ex.mu.Lock()
	for userID, orderbookOrders := range ex.Orders {
		for i := 0; i < len(orderbookOrders); i++ {

			// if the order is not filled place it in the map copy
			if !orderbookOrders[i].IsFilled() {
				newOrderMap[userID] = append(newOrderMap[userID], orderbookOrders[i])
			}
		}
	}
	ex.Orders = newOrderMap
	ex.mu.Unlock()

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	ex.mu.Lock()
	ex.Orders[order.UserID] = append(ex.Orders[order.UserID], order)
	ex.mu.Unlock()

	return nil
}

type PlaceOrderResponse struct {
	OrderID int64
}

// func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
// 	var placeOrderData PlaceOrderRequest

// 	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
// 		return err
// 	}

// 	market := Market(placeOrderData.Market)
// 	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID, placeOrderData.Leverage)

// 	// Limit orders
// 	if placeOrderData.Type == LimitOrder {
// 		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
// 			return err
// 		}

// 	}
// 	// Market orders
// 	if placeOrderData.Type == MarketOrder {
// 		matches, _ := ex.handlePlaceMarketOrder(market, order)

// 		if err := ex.handleMatches(matches); err != nil {
// 			return err
// 		}

// 	}

//		resp := &PlaceOrderResponse{
//			OrderID: order.ID,
//		}
//		return c.JSON(200, resp)
//	}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	req := new(PlaceOrderRequest)
	if err := c.Bind(req); err != nil {
		return err
	}

	// Perform the check before placing the order
	if err := ex.handleCheckMaxContractSize(req.UserID, req.Market, req.Size); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}

	// If the check passes, create the order and add it to the orderbook
	order := orderbook.NewOrder(req.Bid, req.Size, req.UserID, req.Leverage)

	if req.Type == MarketOrder {
		matches, _ := ex.handlePlaceMarketOrder(req.Market, order)
		if err := ex.handleMatches(matches); err != nil {
			return err
		}
	} else if req.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(req.Market, req.Price, order); err != nil {
			return err
		}
	}

	resp := &PlaceOrderResponse{
		OrderID: order.ID,
	}
	return c.JSON(200, resp)
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	// Assume a default user (could be your margin user) to receive the fees
	feeRecipientUser, ok := ex.Users[2]
	if !ok {
		return fmt.Errorf("fee recipient user not found")
	}

	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Ask.UserID)
		}

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Bid.UserID)
		}

		// Calculate the fee from the trade amount
		tradeAmount := match.SizeFilled * match.Ask.Leverage // assuming this is the amount of the trade
		fee := tradeAmount * 0.01                            // taking a 1% fee for example

		// Let's log the status before the trade
		logrus.WithFields(logrus.Fields{
			"fromUserBalance": fromUser.Balance["ETH"],
			"toUserBalance":   toUser.Balance["ETH"],
			"tradeAmount":     tradeAmount,
		}).Info("Before trade")

		// Let the users handle their trades
		fromUser.HandleTrade("ETH", match.SizeFilled, false)
		toUser.HandleTrade("ETH", match.SizeFilled, true)

		// Deduct the fee from the users
		fromUser.Balance["ETH"] -= fee
		toUser.Balance["ETH"] -= fee

		// Add the fee to the fee recipient user's balance
		feeRecipientUser.Balance["ETH"] += fee * 2 // because we took fees from both users

		// Let's log the status after the trade
		logrus.WithFields(logrus.Fields{
			"fromUserBalance": fromUser.Balance["ETH"],
			"toUserBalance":   toUser.Balance["ETH"],
			"fee recipient":   feeRecipientUser.Balance["ETH"],
		}).Info("After trade")
	}
	return nil
}
