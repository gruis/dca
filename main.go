package main

import (
	"fmt"

	"github.com/Rhymond/go-money"
)

type Quote struct {
	Symbol string
	Open   *money.Money
	High   *money.Money
	Low    *money.Money
	Close  *money.Money
}

func (q Quote) Price() *money.Money {
	sum, _ := q.Open.Add(q.Close)
	buckets, _ := sum.Allocate(50, 50)
	return buckets[0]
}

type Bot struct {
	Symbol             string
	TargetValue        *money.Money
	DailyBuyLimitPerc  float64
	DailySellLimitPerc float64
	TotalBuyLimitPerc  float64
	MinProfitPerc      float64

	AssetAmount float64

	BoughtAmount *money.Money
}

func NewBot(b Bot) *Bot {
	b.BoughtAmount = money.New(0, "USD")
	return &b
}

func (b Bot) percOfTarget(v float64) *money.Money {
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

func (b Bot) MinProfit() *money.Money {
	return b.percOfTarget(b.MinProfitPerc)
}

func (b Bot) MinSellValue() *money.Money {
	v, _ := b.TargetValue.Add(b.MinProfit())
	return v
}

func (b Bot) TotalBuyLimit() *money.Money {
	return b.percOfTarget(b.TotalBuyLimitPerc)
}

func (b Bot) DailySellLimit() *money.Money {
	return b.percOfTarget(b.DailySellLimitPerc)
}

func (b Bot) DailyBuyLimit() *money.Money {
	return b.percOfTarget(b.DailyBuyLimitPerc)
}

func (b Bot) Print() {
	fmt.Printf("TargetValue: %s\nMinProfitPerc: %f\n", b.TargetValue.Display(), b.MinProfitPerc)
	fmt.Printf("MinProfit: %s\n", b.MinProfit().Display())
	fmt.Printf("MinSellValue: %s\n", b.MinSellValue().Display())
	fmt.Printf("TotalBuyLimit: %s\n", b.TotalBuyLimit().Display())
	fmt.Printf("DailySellLimit: %s\n", b.DailySellLimit().Display())
	fmt.Printf("DailyBuyLimit: %s\n", b.DailyBuyLimit().Display())
	fmt.Printf("BoughtAmount: %s\n", b.BoughtAmount.Display())
	fmt.Println("")
}

func (b Bot) AssetValue(price *money.Money) *money.Money {
	v := price.AsMajorUnits() * b.AssetAmount
	return money.New(int64(v*100), "USD")
}

func (b *Bot) Process(q Quote) {
	fmt.Printf("Process %s %s; current value: %s\n", q.Price().Display(), q.Symbol, b.AssetValue(q.Price()).Display())
	if yes, _ := b.AssetValue(q.Price()).Equals(b.TargetValue); yes {
		fmt.Println("do nothing")
		return
	}
	if yes, _ := b.AssetValue(q.Price()).LessThan(b.TargetValue); yes {
		// TODO: if buy value is less than fees, skip
		b.buy(q)
	} else {
		b.sell(q)
	}
	fmt.Printf(" Balances\n asset value: %s\n asset amount: %f\n bought amount: %s\n", b.AssetValue(q.Price()).Display(), b.AssetAmount, b.BoughtAmount.Display())
}

func (b *Bot) buy(q Quote) {
	if less, _ := b.BoughtAmount.LessThan(b.TotalBuyLimit()); !less {
		fmt.Printf(" refusing to buy as bought %s exceeds or equals buy limit %s", b.BoughtAmount.Display(), b.TotalBuyLimit().Display())
		return
	}
	var v *money.Money
	d, _ := b.TargetValue.Subtract(b.AssetValue(q.Price()))
	if yes, _ := d.LessThanOrEqual(b.DailyBuyLimit()); yes {
		v = d
	} else {
		v = b.DailyBuyLimit()
	}

	newAssetValue, _ := b.AssetValue(q.Price()).Add(v)
	if less, _ := newAssetValue.LessThanOrEqual(b.TotalBuyLimit()); !less {
		v, _ = b.TotalBuyLimit().Subtract(b.AssetValue(q.Price()))
	}

	amount := v.AsMajorUnits() / q.Price().AsMajorUnits()
	b.doBuy(amount, v)
}

func (b *Bot) doBuy(amount float64, value *money.Money) {
	fmt.Printf(" Buy %f (%s) of %s\n", amount, value.Display(), b.Symbol)
	b.AssetAmount = b.AssetAmount + amount
	t, _ := b.BoughtAmount.Add(value)
	b.BoughtAmount = t
}

func (b *Bot) sell(q Quote) {
	// TODO: if sell value is less than fees, skip
	// 			 if sell value is less than minimum profit, skip

	var v *money.Money
	d, _ := b.AssetValue(q.Price()).Subtract(b.TargetValue)
	if yes, _ := d.LessThanOrEqual(b.DailySellLimit()); yes {
		v = d
	} else {
		v = b.DailySellLimit()
	}

	amount := v.AsMajorUnits() / q.Price().AsMajorUnits()
	b.doSell(amount, v)
}

func (b *Bot) doSell(amount float64, value *money.Money) {
	fmt.Printf(" Sell %f (%s) of %s\n", amount, value.Display(), b.Symbol)
	b.AssetAmount = b.AssetAmount - amount
	t, _ := b.BoughtAmount.Subtract(value)
	b.BoughtAmount = t
}

func main() {
	var bot *Bot

	bot = NewBot(Bot{
		TargetValue:        money.New(50028, "USD"),
		MinProfitPerc:      200,
		TotalBuyLimitPerc:  200,
		DailyBuyLimitPerc:  0.10,
		DailySellLimitPerc: 0.10,
		Symbol:             "SOL",
	})
	bot.Print()

	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(4270, "USD"), Close: money.New(4104, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(4340, "USD"), Close: money.New(4266, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(4690, "USD"), Close: money.New(4510, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(4666, "USD"), Close: money.New(5591, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(5609, "USD"), Close: money.New(3511, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3504, "USD"), Close: money.New(4474, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(4445, "USD"), Close: money.New(3881, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3899, "USD"), Close: money.New(3512, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3123, "USD"), Close: money.New(2469, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(2450, "USD"), Close: money.New(3128, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3140, "USD"), Close: money.New(3000, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3004, "USD"), Close: money.New(3554, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3548, "USD"), Close: money.New(3358, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(3368, "USD"), Close: money.New(2904, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(2900, "USD"), Close: money.New(2738, "USD")})
	bot.Process(Quote{Symbol: "SOLUSDT", Open: money.New(2741, "USD"), Close: money.New(2860, "USD")})
}
