package main

import (
	"fmt"

	"github.com/Rhymond/go-money"
)

type Bot struct {
	TargetValue        *money.Money
	DailyBuyLimitPerc  float64
	DailySellLimitPerc float64
	TotalBuyLimitPerc  float64
	MinProfitPerc      float64
}

func (b Bot) percOfTarget(v float64) *money.Money {
	var p int
	if v > 1 {
		p = int(v)
	} else {
		p = int(v * 100)
	}
	r := 100 - p
	//logrus.WithFields(logrus.Fields{"v": v, "p": p, "r": r}).Info("percOfTarget")
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
	fmt.Println("")
}

func main() {
	var bot Bot

	bot = Bot{
		TargetValue:        money.New(50028, "USD"),
		MinProfitPerc:      200,
		TotalBuyLimitPerc:  200,
		DailyBuyLimitPerc:  0.10,
		DailySellLimitPerc: 0.10,
	}
	bot.Print()

	bot = Bot{
		TargetValue:        money.New(50029, "USD"),
		MinProfitPerc:      0.33,
		TotalBuyLimitPerc:  200,
		DailyBuyLimitPerc:  0.10,
		DailySellLimitPerc: 0.10,
	}
	bot.Print()

	bot = Bot{
		TargetValue:        money.New(50030, "USD"),
		MinProfitPerc:      233,
		TotalBuyLimitPerc:  200,
		DailyBuyLimitPerc:  0.10,
		DailySellLimitPerc: 0.10,
	}
	bot.Print()
}
