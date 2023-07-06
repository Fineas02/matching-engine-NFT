package margin

type MarketConfig struct {
	InitialMarginRequirement float64
	MaximumLeverage          float64
	MaintenanceMargin        float64
	TickSize                 float64
	MinOrder                 float64
	QuantityStep             float64
}
