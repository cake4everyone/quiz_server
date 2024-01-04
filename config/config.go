package config

import (
	"bytes"
	logger "log"
	"os"

	"github.com/spf13/viper"
)

const CONFIGFILE = "config.yaml"

var log = logger.New(logger.Writer(), "[CONFIG] ", logger.LstdFlags|logger.Lmsgprefix)

func Load() {
	viper.SetConfigFile(CONFIGFILE)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	LoadDefaultConfig()

	err := viper.MergeInConfig()
	if os.IsNotExist(err) {
		writeDefaultConfig()
	} else if err != nil {
		panic("Error loading config: " + err.Error())
	}
	log.Println("Successfully loaded config!")
}

// LoadLoadDefaultConfig sets all keys to their default value while also removing all custom
// keys/value pairs.
//
// It panics if an error occours while parsing the config.
func LoadDefaultConfig() {
	err := viper.ReadConfig(bytes.NewBufferString(DEFAULTCONFIG))
	if err != nil {
		panic("Failed to read in default config file. This IS a bug and should be reported\n" + err.Error())
	}
}

func writeDefaultConfig() {
	err := os.WriteFile(CONFIGFILE, []byte(DEFAULTCONFIG), 0644)
	if err != nil {
		panic("Error could not write file: " + err.Error())
	}
}
