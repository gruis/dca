package main

import (
	"fmt"
	"math"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/gruis/dca/config"
	"github.com/gruis/dca/runner"
	"github.com/gruis/dca/strategy"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const AppName = "dca"

type AppConfig struct {
	Bots       []botConfig      `mapstructure:"bots"`
	Currencies []currencyConfig `mapstructure:"currencies"`
}

type currencyConfig struct {
	Code      string `mapstructure:"code"`
	Symbol    string `mapstructure:"symbol"`
	Precision int    `mapstructure:"precision"`
}

type botConfig struct {
	Src             string  `mapstructure:"src"`
	Dst             string  `mapstructure:"dst"`
	ID              string  `mapstructure:"name"`
	TargetValue     float64 `mapstructure:"target-value"`
	MinProfit       float64 `mapstructure:"min-profit"`
	TotalBuyLimit   float64 `mapstructure:"total-buy-limit"`
	SingleBuyLimit  float64 `mapstructure:"single-buy-limit"`
	SingleSellLimit float64 `mapstructure:"single-sell-limit"`
	Span            int     `mapstructure:"span"`
	SpanUnit        string  `mapstructure:"span-unit"`
}

func (bc botConfig) Name() string {
	if bc.ID != "" {
		return bc.ID
	}
	return bc.Dst + bc.Src
}

func (bc *botConfig) Setup() {
	if bc.Src == "" {
		bc.Src = viper.GetString("src")
	}
	if bc.Dst == "" {
		bc.Dst = viper.GetString("dst")
	}

	if bc.TargetValue == 0 {
		bc.TargetValue = viper.GetFloat64("target-value")
	}

	if bc.MinProfit == 0 {
		bc.MinProfit = viper.GetFloat64("min-profit")
	}

	if bc.TotalBuyLimit == 0 {
		bc.TotalBuyLimit = viper.GetFloat64("total-buy-limit")
	}

	if bc.SingleBuyLimit == 0 {
		bc.SingleBuyLimit = viper.GetFloat64("single-buy-limit")
	}

	if bc.SingleSellLimit == 0 {
		bc.SingleSellLimit = viper.GetFloat64("single-sell-limit")
	}

	if bc.SpanUnit == "" {
		bc.SpanUnit = viper.GetString("span-unit")
	}

	if bc.Span == 0 {
		bc.Span = viper.GetInt("span")
	}
}

func (bc *botConfig) ToDCA() (b *strategy.DCA) {
	bc.Setup()
	srcCurrency := money.GetCurrency(bc.Src)
	dstCurrency := money.GetCurrency(bc.Dst)
	if srcCurrency == nil {
		panic(fmt.Sprintf("unknown currency: %s", bc.Src))
	}
	if dstCurrency == nil {
		panic(fmt.Sprintf("unknown currency: %s", bc.Dst))
	}

	targetValue := bc.TargetValue * math.Pow10(srcCurrency.Fraction)

	b = strategy.NewDCA(strategy.DCA{
		Currency:            srcCurrency,
		Target:              dstCurrency,
		TargetValue:         money.New(int64(targetValue), srcCurrency.Code),
		MinProfitPerc:       bc.MinProfit,
		TotalBuyLimitPerc:   bc.TotalBuyLimit,
		SingleBuyLimitPerc:  bc.SingleBuyLimit,
		SingleSellLimitPerc: bc.SingleSellLimit,
		MinTransactionSpan:  bc.minTransactionSpan(),
		PrintTransactions:   viper.GetBool("show-transactions"),
	})
	return b
}

func (bc botConfig) minTransactionSpan() (minTransactionSpan time.Duration) {
	s := time.Duration(bc.Span)
	switch bc.SpanUnit {
	case "seconds":
		minTransactionSpan = time.Second * s
	case "minutes":
		minTransactionSpan = time.Minute * s
	case "hours":
		minTransactionSpan = time.Hour * s
	case "days":
		minTransactionSpan = (time.Hour * 24) * s
	case "weeks":
		minTransactionSpan = (time.Hour * 24 * 7) * s
	default:
		panic(fmt.Sprintf("unrecognized span type: %s", bc.SpanUnit))
	}

	return
}

func addCurrencies(config AppConfig) {
	for _, c := range config.Currencies {
		money.AddCurrency(c.Code, c.Symbol, "$1", ".", ",", c.Precision)
	}
}

func init() {
	config.AddString("src", "USDT", "The base currency the bot will use for investing")
	config.AddString("dst", "BTC", "The currency the bot will invest in")
	config.AddFloat64("target-value", 1_000, "The amount of value (in src) of dst currency to maintain")
	config.AddInt("span", 4, "minimum number of span units between transations")
	config.AddString("span-unit", "days", "type of transaction span: seconds, minutes, hours, days, weeks")
	config.AddFloat64("min-profit", 200, "profit (percentage) threshold before selling")
	config.AddFloat64("total-buy-limit", 200, "percentage of target value that can be bought at any one time")
	config.AddFloat64("single-buy-limit", 0.10, "percentage of target value that can be used for a single purchase")
	config.AddFloat64("single-sell-limit", 0.10, "percentage of target value that can be sold at in a single transaction")
	config.AddBool("show-transactions", false, "print a csv of all transactions")
}

func main() {
	log.SetLevel(log.WarnLevel)
	config.Load(AppName)
	args := flag.Args()
	log.WithField("commands", args).Debug("command line parsed")

	var appConfig AppConfig
	if err := viper.Unmarshal(&appConfig); err != nil {
		panic(fmt.Errorf("cannot parse configuration: %v", err))
	}
	addCurrencies(appConfig)

	var dca *strategy.DCA
	var bc *botConfig

	if len(args) == 0 {
		// Will result in a bot constructed via command line arguments
		bc = &botConfig{}
	} else {
		for _, b := range appConfig.Bots {
			if b.Name() == args[0] {
				bc = &b
				break
			}
		}
		if bc == nil {
			panic(fmt.Sprintf("unknown bot: %s", args[0]))
		}
	}

	bc.Setup()
	log.WithField("bot config", bc).Warn("chose bot config")

	dca = bc.ToDCA()
	dca.PrintTransactions = viper.GetBool("show-transactions")

	r := runner.BinanceFile{Src: dca.Target, Dst: dca.Currency}
	err := dca.Watch(r)
	dca.Print()

	if err != nil {
		panic(err)
	}
}
