package margin

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type Position struct {
	Asset            string
	Side             string
	Leverage         float64
	Size             float64
	OpenPrice        float64
	UnrealizedPNL    float64
	RealizedPNL      float64
	LiquidationPrice float64
}

type User struct {
	ID            int64
	Balance       map[string]float64
	Positions     []Position
	UnrealizedPNL float64
	RealizedPNL   float64
	Fees          float64
	Equity        float64
}

func NewUser(id int64) *User {
	return &User{
		ID:        id,
		Balance:   map[string]float64{"ETH": 1000},
		Positions: []Position{},
		Equity:    1000,
	}
}

func (u *User) HandleTrade(asset string, size float64, leverage float64, isBuyer bool) {
	// Update user's positions based on the trade
	position := Position{
		Asset: asset,
		Size:  size * leverage, // The position size is multiplied by the leverage
	}

	u.Positions = append(u.Positions, position)

	// If user is a buyer, the cost of the trade is deducted from user's balance
	// If user is a seller, the revenue from the trade is added to user's balance
	if isBuyer {
		// For a buyer, the cost of the trade is size
		u.Balance[asset] -= size
	} else {
		// For a seller, the revenue from the trade is size
		u.Balance[asset] += size
	}

	// Log the updated user state
	logrus.WithFields(logrus.Fields{
		"userID":       u.ID,
		"asset":        asset,
		"balance":      u.Balance[asset],
		"leverage":     leverage,
		"tradeSize":    size,
		"positionSize": position.Size,
	}).Info("updated user state after trade")
}

func (u *User) CalculatePotentialLeverage(size float64, price float64, marketConfig *MarketConfig) error {
	maxContractSize := (u.Balance["ETH"] * marketConfig.MaximumLeverage) / price // assuming ETH as asset for example

	if size > maxContractSize {
		return fmt.Errorf("order size too large: maxContractSize %f, order size %f", maxContractSize, size)
	}

	return nil
}

func (u *User) UpdateEquity() float64 {
	return u.Balance["ETH"] + u.UnrealizedPNL + u.RealizedPNL - u.Fees
}
