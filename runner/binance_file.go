package runner

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gruis/dca/prices"
	"github.com/gruis/dca/strategy"
)

// BinanceFile is responsible for simulating quote streams
type BinanceFile struct {
	Symbol string
}

func (r BinanceFile) streamKlines(handler func([]interface{}) error) error {
	var err error
	f, err := os.Open(fmt.Sprintf("%s.json", r.Symbol))
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// We expect an Array of Arrays, so a we pull of the token that starts the
	// wrapping Array and do a quick sanity check to verify that it starts an
	// Array
	t, err := dec.Token()
	if err != nil {
		return err
	}
	d, dok := t.(json.Delim)
	if !dok || d.String() != "[" {
		return fmt.Errorf("%v: expected '[' (string), got '%v' (%T)", KlineParseError, t, t)
	}

	for dec.More() {
		var kline []interface{}
		if err := dec.Decode(&kline); err != nil {
			return err
		}
		if err := handler(kline); err != nil {
			return err
		}
	}

	t, err = dec.Token()
	if err != nil {
		return err
	}
	d, dok = t.(json.Delim)
	if !dok || d.String() != "]" {
		return fmt.Errorf("%v: expected ']' (string), got '%v' (%T)", KlineParseError, t, t)
	}

	return nil
}

// Stream is the primary interface into the simulated quote data. It provides a
// stream of quotes to a callback function.
func (r BinanceFile) Stream(handler func(strategy.Quote) error) error {
	var err error
	err = r.streamKlines(func(k []interface{}) error {
		q := prices.NewKline(r.Symbol)
		if err = q.FromBinance(k); err != nil {
			return err
		}
		if err = handler(*q); err != nil {
			return err
		}
		return nil
	})
	return err
}
