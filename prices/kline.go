package prices

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Rhymond/go-money"
)

type Kline struct {
	symbol    string
	Open      *money.Money
	High      *money.Money
	Low       *money.Money
	Close     *money.Money
	OpenTime  time.Time
	CloseTime time.Time
}

func NewKline(symbol string) *Kline {
	return &Kline{symbol: symbol}
}

func (q Kline) Symbol() string {
	return q.symbol
}

// Price is the average price during the time window
func (q Kline) Price() *money.Money {
	sum, _ := q.Open.Add(q.Close)
	buckets, _ := sum.Allocate(50, 50)
	return buckets[0]
}

// Time is the mid-point during the quote time window
func (q Kline) Time() time.Time {
	return q.OpenTime.Add(q.CloseTime.Sub(q.OpenTime) / 2)
}

var BinanceDataFormatError = errors.New("Binance data is improperly formatted")

// FromBinance extract Kline data from a Binance API single Kline data response
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
func (q *Kline) FromBinance(data []interface{}) error {
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

func (q Kline) MoneyFor(s, currency string) (*money.Money, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return money.New(int64(v*100), currency), nil
}

func (q Kline) Print() {
	fmt.Printf("%s - open: %s (@%s), close: %s (@%s)\n",
		q.Symbol(), q.Open.Display(), q.OpenTime.UTC(), q.Close.Display(), q.CloseTime.UTC(),
	)
}
