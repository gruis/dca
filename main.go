package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/gruis/dca/bot"
	log "github.com/sirupsen/logrus"
)

type Quote struct {
	symbol    string
	Open      *money.Money
	High      *money.Money
	Low       *money.Money
	Close     *money.Money
	OpenTime  time.Time
	CloseTime time.Time
}

func (q Quote) Symbol() string {
	return q.symbol
}

// Price is the average price during the time window
func (q Quote) Price() *money.Money {
	sum, _ := q.Open.Add(q.Close)
	buckets, _ := sum.Allocate(50, 50)
	return buckets[0]
}

// Time is the mid-point during the quote time window
func (q Quote) Time() time.Time {
	return q.OpenTime.Add(q.CloseTime.Sub(q.OpenTime) / 2)
}

var BinanceDataFormatError = errors.New("Binance data is improperly formatted")

// [
//   [
//     1499040000000,      // Open time
//     "0.01634790",       // Open
//     "0.80000000",       // High
//     "0.01575800",       // Low
//     "0.01577100",       // Close
//     "148976.11427815",  // Volume
//     1499644799999,      // Close time
//     "2434.19055334",    // Quote asset volume
//     308,                // Number of trades
//     "1756.87402397",    // Taker buy base asset volume
//     "28.46694368",      // Taker buy quote asset volume
//     "17928899.62484339" // Ignore.
//   ]
// ]

func (q *Quote) FromBinance(data []interface{}) error {
	if len(data) != 12 {
		return BinanceDataFormatError
	}
	q.Open, _ = q.MoneyFor(data[1].(string), "USD")
	q.High, _ = q.MoneyFor(data[2].(string), "USD")
	q.Low, _ = q.MoneyFor(data[3].(string), "USD")
	q.Close, _ = q.MoneyFor(data[4].(string), "USD")
	q.OpenTime = time.Unix(int64((data[0].(float64))/1000), 0)
	q.CloseTime = time.Unix(int64((data[6].(float64))/1000), 0)

	return nil
}

func (q Quote) MoneyFor(s, currency string) (*money.Money, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return money.New(int64(v*100), currency), nil
}

func (q Quote) Print() {
	fmt.Printf("%s - open: %s (@%s), close: %s (@%s)\n",
		q.Symbol(), q.Open.Display(), q.OpenTime.UTC(), q.Close.Display(), q.CloseTime.UTC(),
	)
}

func getKlines(symbol string) [][]interface{} {
	f, err := os.Open(fmt.Sprintf("%s.json", symbol))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	bytes, _ := ioutil.ReadAll(f)
	prices := [][]interface{}{}
	json.Unmarshal(bytes, &prices)
	return prices
}

func getHistory(symbol string) ([]Quote, error) {
	quotes := []Quote{}
	klines := getKlines(symbol)
	for _, p := range klines {
		q := Quote{symbol: symbol}
		if err := q.FromBinance(p); err != nil {
			return quotes, err
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

func main() {
	log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.DebugLevel)
	var b *bot.Strategy
	targetValue := 1_000
	minTransactionSpan := time.Hour * 24 * 4

	b = bot.New(bot.Strategy{
		Currency:           "USD",
		TargetValue:        money.New(int64(targetValue*100), "USD"),
		MinProfitPerc:      200,
		TotalBuyLimitPerc:  200,
		DailyBuyLimitPerc:  0.10,
		DailySellLimitPerc: 0.10,
		Symbol:             "SOL",
		MinTransactionSpan: minTransactionSpan,
	})
	//b.Print()

	history, err := getHistory("SOLUSDT")
	if err != nil {
		panic(err)
	}
	fmt.Println(
		"date, price, " +
			"transaction amount, transaction value, " +
			"asset amount, asset value, " +
			"cash, total value, " +
			"ROI, ROI %, " +
			"num bought, amount bought, value bought, " +
			"num sold, amount sold, value sold",
	)
	for _, quote := range history {
		//quote.Print()
		b.Process(quote)
	}
}
