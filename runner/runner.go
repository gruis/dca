package runner

import (
	"errors"

	"github.com/gruis/dca/strategy"
)

type Runner interface {
	Stream(func(strategy.Quote) error) error
}

var KlineParseError = errors.New("kline data cannot be parsed")
