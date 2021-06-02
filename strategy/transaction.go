package strategy

import (
	"time"

	"github.com/Rhymond/go-money"
)

type Transaction struct {
	Amount float64
	Value  *money.Money
	Fee    *money.Money
	Time   time.Time
}
