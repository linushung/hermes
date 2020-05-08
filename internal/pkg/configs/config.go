package configs

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var instance *viper.Viper

func InitConfig() {
	// default config
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("default")
	v.AddConfigPath("./configs")
	// bind env variable and modify key mapping(e.g. envKey "SYSTEM_PORT" in k8s yaml mapping to configKey "system.port")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("***** [CONFIG][FAIL] ***** Failed to parse system configuration: \n%s", err)
		os.Exit(1)
	}

	instance = v
	log.Infof("***** [INIT:CONFIG] ***** Initialise system configuration ......")
}

// IsConfigSet checks if the key has been set in the configuration
func IsConfigSet(key string) bool {
	return instance.IsSet(key)
}

// GetConfigStr return string value of configuration
func GetConfigStr(key string) string {
	if key != "" {
		return instance.GetString(key)
	}
	return ""
}

// GetConfigBool return boolean value of configuration
func GetConfigBool(key string) bool {
	if key != "" {
		return instance.GetBool(key)
	}
	return false
}

// GetConfigInt return integer value of configuration
func GetConfigInt(key string) int {
	if key != "" {
		return instance.GetInt(key)
	}
	return 0
}

// GetConfigSlice return slice of string value of configuration
func GetConfigSlice(key string) []string {
	if key != "" {
		return instance.GetStringSlice(key)
	}
	return nil
}

// GetConfigMap return map value of configuration
func GetConfigMap(key string) map[string]interface{} {
	if key != "" {
		return instance.GetStringMap(key)
	}
	return nil
}

// GetConfigMapString return map of string value of configuration
func GetConfigMapString(key string) map[string]string {
	if key != "" {
		return instance.GetStringMapString(key)
	}
	return nil
}

// GetConfigUnmarshalKey take a single key and unmarshals it into a Struct
func GetConfigUnmarshalKey(key string, s interface{}) error {
	if key != "" {
		return instance.UnmarshalKey(key, s)
	}
	return nil
}
