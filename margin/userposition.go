package margin

type Position struct {
	AssetName                 string
	Size                      float64
	Price                     float64
	InitialMarginFraction     float64
	MaintenanceMarginFraction float64
}

type User struct {
	ID        string
	Balances  map[string]float64
	Positions []Position
}

func NewUser(id string) *User {
	return &User{
		ID:        id,
		Balances:  make(map[string]float64),
		Positions: []Position{},
	}
}
