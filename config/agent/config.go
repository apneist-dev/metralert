package agentconfig

import (
	"errors"
	"log"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ServerAddress  string
	ReportInterval int
	PollInterval   int
	HashKey        string
	RateLimit      int
	CryptoKey      string
	ConfigFile     string
}

func (cfg *Config) GetConfig() error {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	//defaults
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.SetDefault("address", "localhost:8080")
	viper.SetDefault("report-interval", 10)
	viper.SetDefault("poll-interval", 2)
	viper.SetDefault("rate-limit", 5)

	//flags
	flag.StringP("address", "a", "", "server url")
	flag.IntP("report-interval", "r", 0, "reportInterval")
	flag.IntP("poll-interval", "p", 0, "pollInterval")
	flag.StringP("key", "k", "", "hash key")
	flag.IntP("rate-limit", "l", 0, "rate limit")
	flag.String("crypto-key", "", "Public Key")
	flag.StringP("config", "c", "", "configuration file")
	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		sugar.Warnln("unable to bind flags:", err)
	}

	// env
	viper.AutomaticEnv()

	// json file
	configFileName := viper.GetString("config")

	if configFileName != "" {
		viper.SetConfigType("json")
		viper.SetConfigName(configFileName)
		viper.AddConfigPath("./config/agent/")

		err = viper.ReadInConfig()
		if err != nil {
			sugar.Warnln("unable to read file", configFileName, err)
			return err
		}
	}

	cfg.ServerAddress = viper.GetString("address")
	cfg.HashKey = viper.GetString("key")
	cfg.RateLimit = viper.GetInt("rate-limit")
	cfg.CryptoKey = viper.GetString("crypto-key")
	cfg.ConfigFile = viper.GetString("config")

	cfg.ReportInterval, err = IntervalNormalize(viper.Get("report-interval"))
	if err != nil {
		return err
	}
	cfg.PollInterval, err = IntervalNormalize(viper.Get("poll-interval"))
	if err != nil {
		return err
	}
	return nil
}

func IntervalNormalize(v any) (int, error) {
	switch v := v.(type) {
	case string:
		vs := strings.TrimSuffix(v, "s")
		vi, err := strconv.Atoi(vs)
		if err != nil {
			return 0, err
		}
		return vi, nil
	case int:
		return v, nil
	}
	return 0, errors.New("unknown type")
}
