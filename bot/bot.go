package bot

import (
	"errors"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
	log "github.com/sirupsen/logrus"
)

var InsufficientBudget = errors.New("budget is insufficient to purchace asset at given price")
var UnknownProcessingError = errors.New("an unknown error occured during processing")

type Transaction struct {
	Amount float64
	Value  *money.Money
	Fee    *money.Money
	Time   time.Time
}

type Quote interface {
	Symbol() string
	Price() *money.Money
	Time() time.Time
}

type Strategy struct {
	Symbol              string
	Currency            string
	TargetValue         *money.Money
	SingleBuyLimitPerc  float64
	SingleSellLimitPerc float64
	TotalBuyLimitPerc   float64
	MinProfitPerc       float64
	MinTransactionSpan  time.Duration

	// move these to a ledger

	// AssetAmount is the number of assets in the ledger
	AssetAmount float64
	// BoughtAmount is the cumulative value of all transactions
	BoughtAmount *money.Money
	// Cash is the amount of money in the ledger; this is not the total value of the ledger
	Cash *money.Money

	LastTransaction *Transaction
	LastActedQuote  *Quote

	BuyCnt  int
	SellCnt int

	BuyValue  *money.Money
	SellValue *money.Money

	BuyAmount  float64
	SellAmount float64
}

func New(b Strategy) *Strategy {
	currency := b.Currency
	if currency == "" {
		currency = "USD"
		b.Currency = currency
	}

	if b.BoughtAmount == nil {
		b.BoughtAmount = money.New(0, b.Currency)
	}

	if b.Cash == nil {
		b.Cash = b.TotalBuyLimit()
	}

	if b.BuyValue == nil {
		b.BuyValue = money.New(0, b.Currency)
	}

	if b.SellValue == nil {
		b.SellValue = money.New(0, b.Currency)
	}

	return &b
}

func (b Strategy) percOfTarget(v float64) *money.Money {
	var p int
	if v > 1 {
		p = int(v)
	} else {
		p = int(v * 100)
	}
	r := 100 - p
	buckets, _ := b.TargetValue.Allocate(p, r)
	return buckets[0]
}

func (b Strategy) MinProfit() *money.Money {
	return b.percOfTarget(b.MinProfitPerc)
}

func (b Strategy) MinSellValue() *money.Money {
	v, _ := b.TargetValue.Add(b.MinProfit())
	return v
}

func (b Strategy) TotalBuyLimit() *money.Money {
	return b.percOfTarget(b.TotalBuyLimitPerc)
}

func (b Strategy) Budget() *money.Money {
	return b.TotalBuyLimit()
}

func (b Strategy) SingleSellLimit() *money.Money {
	return b.percOfTarget(b.SingleSellLimitPerc)
}

func (b Strategy) SingleBuyLimit() *money.Money {
	return b.percOfTarget(b.SingleBuyLimitPerc)
}

func (b Strategy) Print() {
	fmt.Printf("TargetValue: %s\nMinProfitPerc: %f\n", b.TargetValue.Display(), b.MinProfitPerc)
	fmt.Printf("MinProfit: %s\n", b.MinProfit().Display())
	fmt.Printf("MinSellValue: %s\n", b.MinSellValue().Display())
	fmt.Printf("TotalBuyLimit: %s\n", b.TotalBuyLimit().Display())
	fmt.Printf("SingleSellLimit: %s\n", b.SingleSellLimit().Display())
	fmt.Printf("SingleBuyLimit: %s\n", b.SingleBuyLimit().Display())
	fmt.Printf("BoughtAmount: %s\n", b.BoughtAmount.Display())
	fmt.Println("")
}

func (b Strategy) LastActedQuoteTime() time.Time {
	if b.LastActedQuote == nil {
		return time.Time{}
	}
	return (*b.LastActedQuote).Time()
}

func (b Strategy) LastTransactionTime() time.Time {
	if b.LastTransaction == nil {
		return time.Time{}
	}
	return (*b.LastTransaction).Time
}

func (b Strategy) AssetValue(price *money.Money) *money.Money {
	v := price.AsMajorUnits() * b.AssetAmount
	return money.New(int64(v*100), b.Currency)
}

func (b Strategy) TotalValue(price *money.Money) *money.Money {
	v, _ := b.AssetValue(price).Add(b.Cash)
	return v
}

func (b Strategy) Roi(price *money.Money) *money.Money {
	// TODO: should it be total value or the realized value
	v, _ := b.TotalValue(price).Subtract(b.Budget())
	return v
}

func (b Strategy) RoiPerc(price *money.Money) float64 {
	return (b.Roi(price).AsMajorUnits() / b.Budget().AsMajorUnits())
}

func (b *Strategy) Process(q Quote) (*Transaction, error) {
	logger := log.WithFields(log.Fields{
		"price":                 q.Price().Display(),
		"symbol":                q.Symbol(),
		"current value":         b.AssetValue(q.Price()).Display(),
		"last acted quote":      b.LastActedQuoteTime(),
		"last transaction time": b.LastTransactionTime(),
	})
	logger.Debug("process")
	transactionSpan := q.Time().Sub(b.LastActedQuoteTime())
	if transactionSpan < b.MinTransactionSpan {
		logger.WithFields(log.Fields{
			"transaction span": transactionSpan,
			"min span":         b.MinTransactionSpan,
		}).Debug("do nothing - minimum transaction span not reached")
		return nil, nil
	}
	if yes, _ := b.AssetValue(q.Price()).Equals(b.TargetValue); yes {
		logger.Debug("do nothing - asset value equals target value")
		return nil, nil
	}

	var (
		action *Transaction
		err    error
	)

	if yes, _ := b.AssetValue(q.Price()).LessThan(b.TargetValue); yes {
		// TODO: if buy value is less than fees, skip
		action, err = b.buy(q)
	} else {
		action, err = b.sell(q)
	}

	if action != nil {
		b.RecordTransaction(action, &q)
	}

	return action, err
}

func (b *Strategy) RecordTransaction(action *Transaction, q *Quote) error {
	if action == nil {
		return nil
	}
	b.LastActedQuote = q

	b.AssetAmount = b.AssetAmount + action.Amount

	if err := b.AddValue(&b.BoughtAmount, action.Value); err != nil {
		return err
	}

	if err := b.SubtractValue(&b.Cash, action.Value); err != nil {
		return err
	}

	if action.Value.IsNegative() {
		b.SellCnt++
		b.SellAmount = b.SellAmount - action.Amount
		b.SubtractValue(&b.SellValue, action.Value)
	} else {
		b.BuyCnt++
		b.BuyAmount = b.BuyAmount + action.Amount
		b.AddValue(&b.BuyValue, action.Value)
	}

	b.LastTransaction = action

	return nil
}

func (b Strategy) AddValue(item **money.Money, value *money.Money) error {
	c, err := (*item).Add(value)
	if err != nil {
		return err
	}
	*item = c
	return nil
}

func (b Strategy) SubtractValue(item **money.Money, value *money.Money) error {
	c, err := (*item).Subtract(value)
	if err != nil {
		return err
	}
	*item = c
	return nil
}

func (b *Strategy) buy(q Quote) (*Transaction, error) {
	// TODO: propogate any errors
	if less, _ := b.BoughtAmount.LessThan(b.TotalBuyLimit()); !less {
		log.WithFields(log.Fields{
			"bought amount":  b.BoughtAmount.Display(),
			"total buy limi": b.TotalBuyLimit().Display(),
		}).Warn("refusing to buy as bought exceeds or equals buy limit ")
		return nil, InsufficientBudget
	}
	var v *money.Money
	d, _ := b.TargetValue.Subtract(b.AssetValue(q.Price()))
	if yes, _ := d.LessThanOrEqual(b.SingleBuyLimit()); yes {
		v = d
	} else {
		v = b.SingleBuyLimit()
	}

	newAssetValue, err := b.AssetValue(q.Price()).Add(v)
	if err != nil {
		return nil, err
	}
	// TODO: propogate any errors
	if less, _ := newAssetValue.LessThanOrEqual(b.TotalBuyLimit()); !less {
		v, err = b.TotalBuyLimit().Subtract(b.AssetValue(q.Price()))
		if err != nil {
			return nil, err
		}
	}

	amount := v.AsMajorUnits() / q.Price().AsMajorUnits()
	return b.doBuy(amount, v)
}

func (b *Strategy) doBuy(amount float64, value *money.Money) (*Transaction, error) {
	log.WithFields(log.Fields{"amount": amount, "value": value.Display(), "symbol": b.Symbol}).Debug("execute buy")
	// TODO: factor in transaction fee of 0.1%; reduce value and amount accordingly
	fee := money.New(0, value.Currency().Code)
	return &Transaction{Amount: amount, Value: value, Time: time.Now(), Fee: fee}, nil
}

func (b *Strategy) sell(q Quote) (*Transaction, error) {
	// TODO: if sell value is less than fees, skip
	var v *money.Money
	d, err := b.AssetValue(q.Price()).Subtract(b.TargetValue)
	if err != nil {
		return nil, err
	}
	if less, _ := d.LessThan(b.MinProfit()); less {
		return nil, nil
	}
	// TODO: propogate errors
	if yes, _ := d.LessThanOrEqual(b.SingleSellLimit()); yes {
		v = d
	} else {
		v = b.SingleSellLimit()
	}

	amount := v.AsMajorUnits() / q.Price().AsMajorUnits()
	return b.doSell(amount, v)
}

func (b *Strategy) doSell(amount float64, value *money.Money) (*Transaction, error) {
	log.WithFields(log.Fields{"amount": amount, "value": value.Display(), "symbol": b.Symbol}).Debug("execute sell")
	// TODO: factor in transaction fee of 0.1%; reduce value and amount accordingly
	fee := money.New(0, value.Currency().Code)

	negative, err := money.New(0, b.Currency).Subtract(value)
	return &Transaction{Amount: 0 - amount, Value: negative, Time: time.Now(), Fee: fee}, err
}
