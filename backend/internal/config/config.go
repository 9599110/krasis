package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

type PostgresConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Issuer     string `mapstructure:"issuer"`
	Expiration int    `mapstructure:"expiration"`
}

type OAuthConfig struct {
	GitHub OAuthProviderConfig `mapstructure:"github"`
	Google OAuthProviderConfig `mapstructure:"google"`
}

type OAuthProviderConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

type AIConfig struct {
	Qdrant QdrantConfig `mapstructure:"qdrant"`
}

type QdrantConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	APIKey     string `mapstructure:"api_key"`
	Collection string `mapstructure:"collection"`
}

type StorageConfig struct {
	MinIO  MinIOConfig `mapstructure:"minio"`
	AI     AIConfig    `mapstructure:"ai"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Allow CONFIG_FILE env to override config path
	if cfgFile := os.Getenv("CONFIG_FILE"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.postgres.host", "localhost")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.user", "postgres")
	v.SetDefault("database.postgres.password", "postgres")
	v.SetDefault("database.postgres.dbname", "krasis")
	v.SetDefault("database.postgres.sslmode", "disable")
	v.SetDefault("database.postgres.max_open_conns", 100)
	v.SetDefault("database.postgres.max_idle_conns", 15)
	v.SetDefault("database.postgres.conn_max_lifetime", 300)
	v.SetDefault("database.redis.addr", "localhost:6379")
	v.SetDefault("database.redis.password", "")
	v.SetDefault("database.redis.db", 0)
	v.SetDefault("database.redis.pool_size", 30)
	v.SetDefault("jwt.secret", "change-me")
	v.SetDefault("jwt.issuer", "krasis-api")
	v.SetDefault("jwt.expiration", 604800)
	v.SetDefault("log.level", "debug")
	v.SetDefault("log.format", "json")
	v.SetDefault("storage.minio.endpoint", "localhost:9000")
	v.SetDefault("storage.minio.access_key", "minioadmin")
	v.SetDefault("storage.minio.secret_key", "minioadmin")
	v.SetDefault("storage.minio.bucket", "krasis")
	v.SetDefault("storage.minio.use_ssl", false)
	v.SetDefault("storage.ai.qdrant.endpoint", "http://localhost:6333")
	v.SetDefault("storage.ai.qdrant.api_key", "")
	v.SetDefault("storage.ai.qdrant.collection", "note_chunks")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found; use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
