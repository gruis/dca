package strategy

import (
	"errors"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
	log "github.com/sirupsen/logrus"
)

var InsufficientBudget = errors.New("budget is insufficient to purchace asset at given price")
var UnknownProcessingError = errors.New("an unknown error occured during processing")

type DCA struct {
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

func NewDCA(bot DCA) *DCA {
	currency := bot.Currency
	if currency == "" {
		currency = "USD"
		bot.Currency = currency
	}

	if bot.BoughtAmount == nil {
		bot.BoughtAmount = money.New(0, bot.Currency)
	}

	if bot.Cash == nil {
		bot.Cash = bot.TotalBuyLimit()
	}

	if bot.BuyValue == nil {
		bot.BuyValue = money.New(0, bot.Currency)
	}

	if bot.SellValue == nil {
		bot.SellValue = money.New(0, bot.Currency)
	}

	return &bot
}

type Streamer interface {
	Stream(func(Quote) error) error
}

func (bot *DCA) Watch(s Streamer) error {
	fmt.Println(
		"date, price, " +
			"transaction amount, transaction value, " +
			"asset amount, asset value, " +
			"cash, total value, " +
			"ROI, ROI %, " +
			"num buys, amount bought, value bought, " +
			"num sells, amount sold, value sold",
	)

	return s.Stream(func(quote Quote) error {
		//quote.Print()
		action, err := bot.Process(quote)
		if err != nil {
			return err
		}
		if action != nil {
			transactionAmount := action.Amount
			transactionValue := action.Value.AsMajorUnits()

			fmt.Printf("%s, %f, %f, %f, %f, %f, %f, %f, %f, %f, %d, %f, %f, %d, %f, %f\n",
				quote.Time().UTC(), quote.Price().AsMajorUnits(),
				transactionAmount, transactionValue,
				bot.AssetAmount, bot.AssetValue(quote.Price()).AsMajorUnits(),
				bot.Cash.AsMajorUnits(), bot.TotalValue(quote.Price()).AsMajorUnits(),
				bot.Roi(quote.Price()).AsMajorUnits(), bot.RoiPerc(quote.Price()),
				bot.BuyCnt, bot.BuyAmount, bot.BuyValue.AsMajorUnits(),
				bot.SellCnt, bot.SellAmount, bot.SellValue.AsMajorUnits(),
			)
		}
		return nil
	})
}

func (bot DCA) percOfTarget(v float64) *money.Money {
	var p int
	if v > 1 {
		p = int(v)
	} else {
		p = int(v * 100)
	}
	r := 100 - p
	buckets, _ := bot.TargetValue.Allocate(p, r)
	return buckets[0]
}

func (bot DCA) MinProfit() *money.Money {
	return bot.percOfTarget(bot.MinProfitPerc)
}

func (bot DCA) MinSellValue() *money.Money {
	v, _ := bot.TargetValue.Add(bot.MinProfit())
	return v
}

func (bot DCA) TotalBuyLimit() *money.Money {
	return bot.percOfTarget(bot.TotalBuyLimitPerc)
}

func (bot DCA) Budget() *money.Money {
	return bot.TotalBuyLimit()
}

func (bot DCA) SingleSellLimit() *money.Money {
	return bot.percOfTarget(bot.SingleSellLimitPerc)
}

func (bot DCA) SingleBuyLimit() *money.Money {
	return bot.percOfTarget(bot.SingleBuyLimitPerc)
}

func (bot DCA) Print() {
	fmt.Printf("\nTarget Value: %s\n", bot.TargetValue.Display())
	fmt.Printf("Min Profit%%: %.2f%%\n", bot.MinProfitPerc)
	fmt.Printf("Min Profit: %s\n", bot.MinProfit().Display())
	fmt.Printf("Min Sell Value: %s\n", bot.MinSellValue().Display())
	fmt.Printf("Total Buy Limit: %s\n", bot.TotalBuyLimit().Display())
	fmt.Printf("Single Sell Limit: %s\n", bot.SingleSellLimit().Display())
	fmt.Printf("Single Buy Limit: %s\n", bot.SingleBuyLimit().Display())
	fmt.Printf("Bought Amount: %s\n", bot.BoughtAmount.Display())
	fmt.Println("")

	fmt.Printf("%s amount:%.2f\n", bot.Symbol, bot.AssetAmount)
	if bot.LastActedQuote != nil {
		fmt.Printf("Asset Value: %s\n", bot.AssetValue((*bot.LastActedQuote).Price()).Display())
		fmt.Printf("Cash: %s\n", bot.Cash.Display())
		fmt.Printf("Total Value: %s\n", bot.TotalValue((*bot.LastActedQuote).Price()).Display())
		fmt.Printf("ROI: %s\n", bot.Roi((*bot.LastActedQuote).Price()).Display())
		fmt.Printf("ROI%%: %.2f%%\n", bot.RoiPerc((*bot.LastActedQuote).Price())*100)
	}

	fmt.Printf("Transactions: %d\n", bot.BuyCnt+bot.SellCnt)
	fmt.Printf("Buy Cnt: %d\n", bot.BuyCnt)
	fmt.Printf("Buy Amount: %.2f\n", bot.BuyAmount)
	fmt.Printf("Buy Value: %s\n", bot.BuyValue.Display())

	fmt.Printf("Sell Cnt: %d\n", bot.SellCnt)
	fmt.Printf("Sell Amount: %.2f\n", bot.SellAmount)
	fmt.Printf("Sell Value: %s\n", bot.SellValue.Display())
	fmt.Println("")
}

func (bot DCA) LastActedQuoteTime() time.Time {
	if bot.LastActedQuote == nil {
		return time.Time{}
	}
	return (*bot.LastActedQuote).Time()
}

func (bot DCA) LastTransactionTime() time.Time {
	if bot.LastTransaction == nil {
		return time.Time{}
	}
	return (*bot.LastTransaction).Time
}

func (bot DCA) AssetValue(price *money.Money) *money.Money {
	v := price.AsMajorUnits() * bot.AssetAmount
	return money.New(int64(v*100), bot.Currency)
}

func (bot DCA) TotalValue(price *money.Money) *money.Money {
	v, _ := bot.AssetValue(price).Add(bot.Cash)
	return v
}

func (bot DCA) Roi(price *money.Money) *money.Money {
	// TODO: should it be total value or the realized value
	v, _ := bot.TotalValue(price).Subtract(bot.Budget())
	return v
}

func (bot DCA) RoiPerc(price *money.Money) float64 {
	return (bot.Roi(price).AsMajorUnits() / bot.Budget().AsMajorUnits())
}

// Process along with buy and sell constitute the DCA algorithm
func (bot *DCA) Process(q Quote) (*Transaction, error) {
	logger := log.WithFields(log.Fields{
		"price":                 q.Price().Display(),
		"symbol":                q.Symbol(),
		"current value":         bot.AssetValue(q.Price()).Display(),
		"last acted quote":      bot.LastActedQuoteTime(),
		"last transaction time": bot.LastTransactionTime(),
	})
	logger.Debug("process")

	transactionSpan := q.Time().Sub(bot.LastActedQuoteTime())
	if transactionSpan < bot.MinTransactionSpan {
		logger.WithFields(log.Fields{
			"transaction span": transactionSpan,
			"min span":         bot.MinTransactionSpan,
		}).Debug("do nothing - minimum transaction span not reached")
		return nil, nil
	}
	if yes, _ := bot.AssetValue(q.Price()).Equals(bot.TargetValue); yes {
		logger.Debug("do nothing - asset value equals target value")
		return nil, nil
	}

	var (
		action *Transaction
		err    error
	)

	if yes, _ := bot.AssetValue(q.Price()).LessThan(bot.TargetValue); yes {
		// TODO: if buy value is less than fees, skip
		action, err = bot.buy(q)
	} else {
		action, err = bot.sell(q)
	}

	if action != nil {
		bot.RecordTransaction(action, &q)
	}

	return action, err
}

func (bot *DCA) buy(q Quote) (*Transaction, error) {
	// TODO: propogate any errors
	if less, _ := bot.BoughtAmount.LessThan(bot.TotalBuyLimit()); !less {
		log.WithFields(log.Fields{
			"bought amount":  bot.BoughtAmount.Display(),
			"total buy limi": bot.TotalBuyLimit().Display(),
		}).Warn("refusing to buy as bought exceeds or equals buy limit ")
		return nil, InsufficientBudget
	}
	var v *money.Money
	d, _ := bot.TargetValue.Subtract(bot.AssetValue(q.Price()))
	if yes, _ := d.LessThanOrEqual(bot.SingleBuyLimit()); yes {
		v = d
	} else {
		v = bot.SingleBuyLimit()
	}

	newAssetValue, err := bot.AssetValue(q.Price()).Add(v)
	if err != nil {
		return nil, err
	}
	// TODO: propogate any errors
	if less, _ := newAssetValue.LessThanOrEqual(bot.TotalBuyLimit()); !less {
		v, err = bot.TotalBuyLimit().Subtract(bot.AssetValue(q.Price()))
		if err != nil {
			return nil, err
		}
	}

	amount := v.AsMajorUnits() / q.Price().AsMajorUnits()
	return bot.doBuy(amount, v)
}

func (bot *DCA) doBuy(amount float64, value *money.Money) (*Transaction, error) {
	log.WithFields(log.Fields{"amount": amount, "value": value.Display(), "symbol": bot.Symbol}).Debug("execute buy")
	// TODO: factor in transaction fee of 0.1%; reduce value and amount accordingly
	fee := money.New(0, value.Currency().Code)
	return &Transaction{Amount: amount, Value: value, Time: time.Now(), Fee: fee}, nil
}

func (b *DCA) sell(q Quote) (*Transaction, error) {
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

func (bot *DCA) doSell(amount float64, value *money.Money) (*Transaction, error) {
	log.WithFields(log.Fields{"amount": amount, "value": value.Display(), "symbol": bot.Symbol}).Debug("execute sell")
	// TODO: factor in transaction fee of 0.1%; reduce value and amount accordingly
	fee := money.New(0, value.Currency().Code)

	negative, err := money.New(0, bot.Currency).Subtract(value)
	return &Transaction{Amount: 0 - amount, Value: negative, Time: time.Now(), Fee: fee}, err
}

func (bot *DCA) RecordTransaction(action *Transaction, q *Quote) error {
	if action == nil {
		return nil
	}
	bot.LastActedQuote = q

	bot.AssetAmount = bot.AssetAmount + action.Amount

	if err := bot.AddValue(&bot.BoughtAmount, action.Value); err != nil {
		return err
	}

	if err := bot.SubtractValue(&bot.Cash, action.Value); err != nil {
		return err
	}

	if action.Value.IsNegative() {
		bot.SellCnt++
		bot.SellAmount = bot.SellAmount - action.Amount
		bot.SubtractValue(&bot.SellValue, action.Value)
	} else {
		bot.BuyCnt++
		bot.BuyAmount = bot.BuyAmount + action.Amount
		bot.AddValue(&bot.BuyValue, action.Value)
	}

	bot.LastTransaction = action

	return nil
}

func (bot DCA) AddValue(item **money.Money, value *money.Money) error {
	c, err := (*item).Add(value)
	if err != nil {
		return err
	}
	*item = c
	return nil
}

func (bot DCA) SubtractValue(item **money.Money, value *money.Money) error {
	c, err := (*item).Subtract(value)
	if err != nil {
		return err
	}
	*item = c
	return nil
}
