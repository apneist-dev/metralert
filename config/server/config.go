package config

import (
	"context"
	"errors"
	"log"
	"metralert/internal/metrics"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ServerGRPC      bool
	ServerAddress   string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseAddress string
	HashKey         string
	AuditFile       string
	AuditURL        string
	CryptoKey       string
	ConfigFile      string
	TrustedSubnet   string
	Logger          *zap.SugaredLogger
	Storage         StorageInterface
}

type StorageInterface interface {
	UpdateMetric(ctx context.Context, metric metrics.Metrics) (*metrics.Metrics, error)
	UpdateBatchMetrics(ctx context.Context, metrics []metrics.Metrics) ([]metrics.Metrics, error)
	GetMetricByName(ctx context.Context, metric metrics.Metrics) (*metrics.Metrics, bool)
	GetMetrics(ctx context.Context) (map[string]any, error)
	PingDatabase(ctx context.Context) error
	BackupService(storeInterval int) error
	Shutdown() error
}

func (cfg *Config) GetConfig() error {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	cfg.Logger = logger.Sugar()

	//defaults
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.SetDefault("address", "localhost:8080")
	viper.SetDefault("store-interval", 300)
	viper.SetDefault("server-grpc", true)

	// flags
	flag.StringP("config", "c", "", "json config file")
	flag.StringP("address", "a", "", "server url")
	flag.IntP("store-interval", "i", 300, "file swap interval")
	flag.StringP("file-storage-path", "f", "metrics_database.json", "filename to store metrics")
	flag.BoolP("restore", "r", false, "restore metrics on startup")
	flag.StringP("database-dsn", "d", "", "database dsn")
	flag.StringP("key", "k", "", "hash key")
	flag.String("audit-file", "", "path of a file to store audit logs")
	flag.String("audit-url", "", "path of a file to store audit logs")
	flag.String("crypto-key", "", "private key")
	flag.StringP("trusted-subnet", "t", "", "trusted subnet")
	flag.BoolP("server-grpc", "g", false, "server grpc enable")
	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		cfg.Logger.Warnln("unable to bind flags:", err)
	}

	// env config
	viper.AutomaticEnv()

	// json config
	configFileName := viper.GetString("config")

	if configFileName != "" {
		viper.SetConfigType("json")
		viper.SetConfigName(configFileName)
		viper.AddConfigPath("./config/server/")

		err = viper.ReadInConfig()
		if err != nil {
			cfg.Logger.Warnln("unable to read file", configFileName, err)
			return err
		}
	}

	cfg.ServerGRPC = viper.GetBool("server-grpc")
	cfg.ServerAddress = viper.GetString("address")
	cfg.FileStoragePath = viper.GetString("file-storage-path")
	cfg.Restore = viper.GetBool("restore")
	cfg.DatabaseAddress = viper.GetString("database-dsn")
	cfg.HashKey = viper.GetString("key")
	cfg.AuditFile = viper.GetString("audit-file")
	cfg.AuditURL = viper.GetString("audit-url")
	cfg.CryptoKey = viper.GetString("crypto-key")
	cfg.ConfigFile = viper.GetString("config")
	cfg.TrustedSubnet = viper.GetString("trusted-subnet")

	cfg.StoreInterval, err = IntervalNormalize(viper.GetInt("store-interval"))
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
