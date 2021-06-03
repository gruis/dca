package main

import (
	"time"

	"github.com/Rhymond/go-money"
	"github.com/gruis/dca/runner"
	"github.com/gruis/dca/strategy"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.DebugLevel)
	var b *strategy.DCA
	targetValue := 1_000
	minTransactionSpan := time.Hour * 24 * 4

	r := runner.BinanceFile{Symbol: "SOLUSDT"}
	b = strategy.NewDCA(strategy.DCA{
		Currency:            "USD",
		TargetValue:         money.New(int64(targetValue*100), "USD"),
		MinProfitPerc:       200,
		TotalBuyLimitPerc:   200,
		SingleBuyLimitPerc:  0.10,
		SingleSellLimitPerc: 0.10,
		Symbol:              "SOL",
		MinTransactionSpan:  minTransactionSpan,
	})
	err := b.Watch(r)
	b.Print()

	if err != nil {
		panic(err)
	}
}
