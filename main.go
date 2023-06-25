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
	"github.com/ethereum/go-ethereum/core/types"
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
		Type   OrderType
		Bid    bool
		Size   float64
		Price  float64
		Market Market
	}

	Order struct {
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

func main() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	ex, err := NewExchange(exchangePrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	e.GET("/book/:market", ex.handleGetMarket)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	//adress := common.HexToAddress("0x32309F91b1e1D66776444e1d580eEb7B9a1B8e9f")

	privateKey, err := crypto.HexToECDSA("c57297908760fb07925613d8f57a8e4923a6d946374f7466e55450986b425be6")
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {

		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(1000000000000000000) // in wei (1 eth)

	gasLimit := uint64(21000) // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

	chainID := big.NewInt(1337)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Fatal(err)
	}

	balance, err := client.BalanceAt(ctx, toAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(balance)

	e.Start(":3000")
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	PrivateKey         *ecdsa.PrivateKey
	orderbooks         map[Market]*orderbook.Orderbook
	idToOrderMap       map[int64]*orderbook.Order
	orderIdToMarketMap map[int64]Market
}

func NewExchange(privateKey string) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	idToOrderMap := make(map[int64]*orderbook.Order)
	orderIdToMarketMap := make(map[int64]Market)
	orderbooks[MarketETH] = orderbook.NewOrderbook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &Exchange{
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

	market := Market(placeOrderData.Market)

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

		return c.JSON(200, map[string]any{"matches": matchedOrders})
	}
	return nil
}
