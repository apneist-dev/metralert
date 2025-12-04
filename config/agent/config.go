package agentconfig

import (
	"log"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ServerAddress  string `env:"ADDRESS" mapstructure:"address"`
	ReportInterval int    `env:"REPORT_INTERVAL" mapstructure:"report_interval"`
	PollInterval   int    `env:"POLL_INTERVAL" mapstructure:"poll_interval"`
	HashKey        string `env:"KEY" mapstructure:"-"`
	RateLimit      int    `env:"RATE_LIMIT" mapstructure:"-"`
	CryptoKey      string `env:"CRYPTO_KEY" mapstructure:"crypto_key"`
	ConfigFile     string `env:"CONFIG" mapstructure:"-"`
}

func (cfg *Config) GetConfig() error {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	//defaults
	viper.SetDefault("address", "localhost:8080")
	viper.SetDefault("report_interval", 10)
	viper.SetDefault("poll_interval", 2)
	viper.SetDefault("rate_limit", 5)

	//flags
	flag.StringP("address", "a", "", "server url")
	flag.IntP("report_interval", "r", 0, "reportInterval")
	flag.IntP("poll_interval", "p", 0, "pollInterval")
	flag.StringP("key", "k", "", "hash key")
	flag.IntP("rate_limit", "l", 0, "rate limit")
	flag.String("crypto-key", "", "Public Key")
	flag.Parse()

	viper.BindPFlags(flag.CommandLine)

	// env
	viper.AutomaticEnv()

	// json file
	configFileName := viper.GetString("config")
	if configFileName != "" {
		viper.SetConfigName(configFileName)
		viper.AddConfigPath("./config/agent")

		err = viper.ReadInConfig()
		if err != nil {
			sugar.Warnln("unable to read file", configFileName)
			return err
		}
	}

	cfg.ServerAddress = viper.GetString("address")
	cfg.ReportInterval = viper.GetInt("report_interval")
	cfg.PollInterval = viper.GetInt("poll_interval")
	cfg.HashKey = viper.GetString("key")
	cfg.RateLimit = viper.GetInt("rate_limit")
	cfg.CryptoKey = viper.GetString("crypto_key")
	cfg.ConfigFile = viper.GetString("config")

	return nil
}
