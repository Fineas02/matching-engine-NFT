package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	orderbook "github.com/fineas02/matching-engine/orderbook"
	"github.com/labstack/echo/v4"
)

const (
	exchangePrivateKey           = "c57297908760fb07925613d8f57a8e4923a6d946374f7466e55450986b425be6"
	MarketOrder        OrderType = "MARKET"
	LimitOrder         OrderType = "LIMIT"
	MarketETH          Market    = "ETH"
)

type (
	OrderType string
	Market    string

	PlaceOrderRequest struct {
		UserID int64
		Type   OrderType
		Bid    bool
		Size   float64
		Price  float64
		Market Market
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
		Price float64
		Size  float64
		ID    int64
	}
)

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(privKey string, id int64) *User {
	pk, err := crypto.HexToECDSA(privKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID:         id,
		PrivateKey: pk,
	}
}

func main() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchange(exchangePrivateKey, client)
	if err != nil {
		log.Fatal(err)
	}
	buyerAddressStr := "0x2692d7E65153bA4b876cB05C1e3A37482CBb41eB"
	buyerBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(buyerAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("buyer: ", buyerBalance)

	sellerAddressStr := "0x0771fe554d8B6070d6814b8bF67057890F9Af78C"
	sellerBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(sellerAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("seller: ", sellerBalance)
	johnAddressStr := "0x7Ac34ec4de19539E8EAE363db10F5E68133004c7"
	johnBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(johnAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("John: ", johnBalance)

	pkStr8 := "0662c9ed2db01dfd927ca888c3d35e03f118e9673bc1fdb3d0aeaf65f9d30d48"
	user8 := NewUser(pkStr8, 8)
	ex.Users[user8.ID] = user8

	pkStr7 := "cc23e29b4c15d8e51820813f50fe3db1bd1d147eb285a643ecaf7f63e09abc65"
	user7 := NewUser(pkStr7, 7)
	ex.Users[user7.ID] = user7

	johnPk := "b44a699708ae89aa94cf55736bcd2b32a2257168c337768626b5a7b1b9e9c76d"
	john := NewUser(johnPk, 6)
	ex.Users[john.ID] = john

	e.GET("/book/:market", ex.handleGetMarket)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	e.Start(":3000")
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	Client             *ethclient.Client
	Users              map[int64]*User
	orders             map[int64]int64
	PrivateKey         *ecdsa.PrivateKey
	orderbooks         map[Market]*orderbook.Orderbook
	idToOrderMap       map[int64]*orderbook.Order
	orderIdToMarketMap map[int64]Market
}

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	idToOrderMap := make(map[int64]*orderbook.Order)
	orderIdToMarketMap := make(map[int64]Market)
	orderbooks[MarketETH] = orderbook.NewOrderbook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &Exchange{
		Client:             client,
		Users:              make(map[int64]*User),
		orders:             make(map[int64]int64),
		PrivateKey:         pk,
		orderbooks:         orderbooks,
		idToOrderMap:       idToOrderMap,
		orderIdToMarketMap: orderIdToMarketMap,
	}, nil
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
		ob.CancelOrder(order)
		delete(ex.orderIdToMarketMap, id)
	}

	return c.JSON(http.StatusOK, "Order cancelled successfully")
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

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}

	for i := 0; i < len(matchedOrders); i++ {
		id := matches[i].Bid.ID
		if isBid {
			id = matches[i].Ask.ID
		}

		matchedOrders[i] = &MatchedOrder{
			ID:    id,
			Size:  matches[i].SizeFilled,
			Price: matches[i].Price,
		}
	}

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	return nil
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	ex.idToOrderMap[order.ID] = order
	ex.orderIdToMarketMap[order.ID] = market

	if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}
		return c.JSON(200, map[string]any{"msg": "limit order placed"})
	}
	if placeOrderData.Type == MarketOrder {
		matches, matchedOrders := ex.handlePlaceMarketOrder(market, order)

		if err := ex.handleMatches(matches); err != nil {
			return err
		}

		return c.JSON(200, map[string]any{"matches": matchedOrders})
	}

	return nil
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Ask.UserID)
		}

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Bid.UserID)
		}
		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		// exchangePubKey := ex.PrivateKey.Public()
		// publicKeyECDSA, ok := exchangePubKey.(*ecdsa.PublicKey)
		// if !ok {
		// 	return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		// }

		amount := big.NewInt(int64(match.SizeFilled))

		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}

	return nil
}
