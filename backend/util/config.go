package util

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"reflect"
	"strings"
	"time"
)

type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	LDAPServers          []string      `mapstructure:"LDAP_SERVERS"`
	LDAPPort             int           `mapstructure:"LDAP_PORT"`
	LDAPDomain           string        `mapstructure:"LDAP_DOMAIN"`
	LDAPSecurity         string        `mapstructure:"LDAP_SECURITY"`
	LDAPTimeout          time.Duration `mapstructure:"LDAP_TIMEOUT"`
	LDAPCACert           string        `mapstructure:"LDAP_CA_CERT"`
	ActiveUserStatus     string        `mapstructure:"ACTIVE_USER_STATUS"`
}

func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("app")
	v.SetConfigType("env")
	v.SetDefault("DB_DRIVER", "oracle")
	v.SetDefault("HTTP_SERVER_ADDRESS", ":8080")
	v.SetDefault("ACCESS_TOKEN_DURATION", "15m")
	v.SetDefault("REFRESH_TOKEN_DURATION", "24h")
	v.SetDefault("LDAP_PORT", 636)
	v.SetDefault("LDAP_SECURITY", "ldaps")
	v.SetDefault("LDAP_TIMEOUT", "5s")
	v.SetDefault("ACTIVE_USER_STATUS", "A")
	v.AutomaticEnv()
	typ := reflect.TypeOf(config)
	for i := 0; i < typ.NumField(); i++ {
		if err = v.BindEnv(typ.Field(i).Tag.Get("mapstructure")); err != nil {
			return
		}
	}
	if err = v.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if !errors.As(err, &missing) {
			return
		}
	}
	if err = v.Unmarshal(&config); err != nil {
		return
	}
	config.LDAPServers = strings.FieldsFunc(v.GetString("LDAP_SERVERS"), func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	if config.DBDriver != "oracle" || !strings.HasPrefix(config.DBSource, "oracle://") {
		return config, fmt.Errorf("DB_DRIVER must be oracle and DB_SOURCE must be an oracle:// connection URL")
	}
	if len(config.TokenSymmetricKey) != 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain 32 bytes")
	}
	if len(config.LDAPServers) == 0 || config.LDAPDomain == "" {
		return config, fmt.Errorf("LDAP_SERVERS and LDAP_DOMAIN are required")
	}
	if config.LDAPPort < 1 || config.LDAPPort > 65535 || config.LDAPTimeout <= 0 {
		return config, fmt.Errorf("invalid LDAP port or timeout")
	}
	if config.LDAPSecurity != "ldaps" && config.LDAPSecurity != "starttls" {
		return config, fmt.Errorf("LDAP_SECURITY must be ldaps or starttls")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= config.AccessTokenDuration || config.ActiveUserStatus == "" {
		return config, fmt.Errorf("invalid token durations or ACTIVE_USER_STATUS")
	}
	return config, nil
}
