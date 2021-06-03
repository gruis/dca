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
	"github.com/gruis/dca/strategy"
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

type Runner struct {
	Symbol string
}

type RunnerStreamHandler func(Quote) error

func (r Runner) getKlines() [][]interface{} {
	f, err := os.Open(fmt.Sprintf("%s.json", r.Symbol))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	bytes, _ := ioutil.ReadAll(f)
	prices := [][]interface{}{}
	json.Unmarshal(bytes, &prices)
	return prices
}

func (r Runner) streamKlines(handler func([]interface{}) error) error {
	var err error
	// we are faking a stream here so that upstream code will be forced to deal
	// with streaming apis that they should ultimately integrate with
	// TODO: use a JSON stream parser instead of getKlines
	//       https://golang.org/pkg/encoding/json/#example_Decoder_Decode_stream
	for _, k := range r.getKlines() {
		if err = handler(k); err != nil {
			continue
		}
	}
	return err
}

func (r Runner) getHistory() ([]Quote, error) {
	quotes := []Quote{}
	err := r.Stream(func(q Quote) error {
		quotes = append(quotes, q)
		return nil
	})
	return quotes, err
}

func (r Runner) Stream(handler RunnerStreamHandler) error {
	var err error
	r.streamKlines(func(k []interface{}) error {
		q := Quote{symbol: r.Symbol}
		if err = q.FromBinance(k); err != nil {
			return err
		}
		if err = handler(q); err != nil {
			return err
		}
		return nil
	})
	return err
}

func main() {
	log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.DebugLevel)
	var b *strategy.DCA
	targetValue := 1_000
	minTransactionSpan := time.Hour * 24 * 4

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
	//b.Print()

	fmt.Println(
		"date, price, " +
			"transaction amount, transaction value, " +
			"asset amount, asset value, " +
			"cash, total value, " +
			"ROI, ROI %, " +
			"num buys, amount bought, value bought, " +
			"num sells, amount sold, value sold",
	)
	r := Runner{Symbol: "SOLUSDT"}
	err := r.Stream(func(quote Quote) error {
		//quote.Print()
		action, err := b.Process(quote)
		if err != nil {
			return err
		}
		if action != nil {
			transactionAmount := action.Amount
			transactionValue := action.Value.AsMajorUnits()

			fmt.Printf("%s, %f, %f, %f, %f, %f, %f, %f, %f, %f, %d, %f, %f, %d, %f, %f\n",
				quote.Time().UTC(), quote.Price().AsMajorUnits(),
				transactionAmount, transactionValue,
				b.AssetAmount, b.AssetValue(quote.Price()).AsMajorUnits(),
				b.Cash.AsMajorUnits(), b.TotalValue(quote.Price()).AsMajorUnits(),
				b.Roi(quote.Price()).AsMajorUnits(), b.RoiPerc(quote.Price()),
				b.BuyCnt, b.BuyAmount, b.BuyValue.AsMajorUnits(),
				b.SellCnt, b.SellAmount, b.SellValue.AsMajorUnits(),
			)
		}
		return nil
	})
	b.Print()

	if err != nil {
		panic(err)
	}
}
