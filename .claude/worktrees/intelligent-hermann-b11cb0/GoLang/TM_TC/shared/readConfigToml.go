package shared

import (
	"fmt"

	"github.com/spf13/viper"
)

func ReadTomlConfigFile() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

func GetTomlConfigValue(key string) string {
	return viper.GetString(key)
}
