package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type ChangeHandler func() error

var changeHandlers = map[string]ChangeHandler{}

func OnChange(name string, handler ChangeHandler) {
	log.WithField("handler", name).Info("added changed handler")
	if _, exists := changeHandlers[name]; exists {
		log.WithField("handler", name).Warn("config change handler reassigned")
	}
	changeHandlers[name] = handler
}

func changed() {
	for name, handler := range changeHandlers {
		if err := handler(); err != nil {
			log.WithError(err).
				WithField("handler", name).
				Error("config handler failed")
		}
	}
}

var flagsOnce = sync.Once{}
var defLogConfig = LogConfig{
	Level:  "info",
	JSON:   true,
	Text:   false,
	Output: os.Stdout,
}

func AddStringSlice(name string, defVal []string, help string) {
	flag.StringSlice(name, defVal, help)
}

func AddString(name, defVal, help string) {
	flag.String(name, defVal, help)
}

func AddFloat64(name string, defVal float64, help string) {
	flag.Float64(name, defVal, help)
}

func AddInt(name string, defVal int, help string) {
	flag.Int(name, defVal, help)
}

func AddBool(name string, defVal bool, help string) {
	flag.Bool(name, defVal, help)
}

func AddStringVar(value *string, name, defVal, help string) {
	flag.StringVar(value, name, defVal, help)
}

func addVars() {
	AddString("log-level", defLogConfig.Level, "show logs at or above this level; choices: trace, debug, info, warn, error, fatal, panic")
	AddBool("log-text", false, "log in text format")
	AddBool("log-json", true, "log in json format")
}

//dynConfigFileName allows us to build a configuraiton file name based on
// dynamic components. If any of those dynamic components are empty then they
// will be removed from the final filename
type dynConfigFileName []string

//String assembles a dynConfigFileName into a filename with empty components
// stripped out and all remaining components seperated by '.'
func (c dynConfigFileName) String() string {
	var r dynConfigFileName
	for _, str := range c {
		if str != "" {
			r = append(r, str)
		}
	}
	return strings.Join(r, ".")
}

func parse(name string) error {
	var (
		configFileName  string
		configFilePath  string
		configEnvPrefix string
		env             string
	)

	if len(name) == 0 {
		name = "gruis"
	}

	env = os.Getenv(fmt.Sprintf("%s_ENV", strings.ToUpper(name)))
	if len(env) == 0 {
		env = os.Getenv("ENV")
	}

	defFilename := dynConfigFileName{"config", env}

	flagsOnce.Do(func() {
		// TODO: let `env` above also have the same prefix
		flag.StringVar(&configEnvPrefix, "config-env-prefix", name, "env var name prefix")

		flag.StringVar(&configFileName, "config-name", defFilename.String(), "configuration file name")
		flag.StringVar(&configFilePath, "config-path", relativePath(".."), "directory containing configuration file")

		addVars()
	})

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	flag.Parse()
	viper.SetEnvPrefix(configEnvPrefix)
	viper.AutomaticEnv()

	viper.SetConfigName(configFileName)

	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", name))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", name))

	viper.AddConfigPath(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"file":  viper.ConfigFileUsed(),
		}).Info("Failed to load config file")
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.WithField("file", viper.ConfigFileUsed()).
				Warn("Config file not found")
		} else {
			log.WithError(err).
				WithField("file", viper.ConfigFileUsed()).
				Fatal("Couldn't read config file")
		}
	}

	flag.Parse()

	viper.BindPFlags(flag.CommandLine)
	setLogger()

	OnChange("log", setLogger)

	// Watch for changes to the configuration file and reload the application
	// configuration. Configuration options that are originally specified on
	// the command line will NOT be effected by changes in the config files.
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.WithField("file", e.Name).Warn("Config file changed")
		changed()
	})

	changed()
	return err
}

func relativePath(filename string) string {
	_, dirname, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(dirname), filename)
}

func Load(name string) error {
	defLogConfig.Set()
	return parse(name)
}

//Load direct loads up a configuration from a byte array and triggers any
//subscribed callbacks. It does not merge the configuration with the
//environment or command line variables. This is mostly useful for testing, but
//can be helpful in some other situations.
func LoadDirect(name string, yaml []byte) error {
	defLogConfig.Set()
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer(yaml)); err != nil {
		log.WithError(err).Info("Failed to load config")
		return err
	}
	setLogger()
	changed()
	return nil
}
