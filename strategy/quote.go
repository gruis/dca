package strategy

import (
	"time"

	"github.com/Rhymond/go-money"
)

type Quote interface {
	Symbol() string
	Price() *money.Money
	Time() time.Time
}
