package config

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type LogConfig struct {
	Level  string
	JSON   bool
	Text   bool
	Output io.Writer
}

func (lc LogConfig) SetLevel() {
	level, err := log.ParseLevel(lc.Level)
	if err == nil {
		log.SetLevel(level)
		return
	}

	log.WithError(err).
		WithFields(log.Fields{"level": level, "default": defLogConfig.Level}).
		Info("using default log level")

	level, derr := log.ParseLevel(defLogConfig.Level)
	if derr != nil {
		log.WithError(err).
			WithFields(log.Fields{"level": defLogConfig.Level}).
			Fatal("unable to set log level")
	}

	log.SetLevel(level)
	return
}

func (lc *LogConfig) SetFormat() {
	if lc.JSON {
		log.SetFormatter(&log.JSONFormatter{})
		lc.Text = false
	}
	if lc.Text {
		log.SetFormatter(&log.TextFormatter{})
		lc.JSON = false
	}
}

func (lc LogConfig) Set() {
	lc.SetFormat()
	lc.SetLevel()
	log.SetOutput(lc.Output)

	log.WithFields(
		log.Fields{
			"json":  lc.JSON,
			"text":  lc.Text,
			"level": lc.Level,
		},
	).Info("log configured")
}

// error is for compat with ConfigChangeHandler
func setLogger() error {
	// TODO: update Echo logger
	LogConfig{
		Level:  viper.GetString("log-level"),
		JSON:   viper.GetBool("log-json"),
		Text:   viper.GetBool("log-text"),
		Output: os.Stdout,
	}.Set()
	return nil
}
