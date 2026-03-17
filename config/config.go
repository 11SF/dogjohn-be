package config

import (
	"fmt"
	"log"
	"sync"

	env "github.com/caarlos0/env/v11"
)

// Config is a struct that contains all configuration for the application
// NOTE: struct name should be in lowercase and field name should be in uppercase
// you can group the configuration by adding new struct
// Example:
//
//	type Config struct {
//			...
//			GCP gcp  // no need to add tag `env` for struct here.
//	}
//
// then create gcp struct with tag `env` for each field
//
//	type gcp struct {
//		ProjectID string `env:"GCP_PROJECT_ID"`
//	}
//
// you can add field without grouping them by adding new field with tag `env`
// Example:
//
//	type Config struct {
//		...
//		AppName string `env:"APP_NAME"`
//	}
type Config struct {
	Server        Server
	AccessControl AccessControl
	Header        Header
	Database      Database
	SlipOK        SlipOK
	HomeAssistant HomeAssistant

	IsByPassVerifySlip bool `env:"FEATURE_FLAG_IS_BYPASS_VERIFY_SLIP,envDefault=false"`
}

type Server struct {
	Hostname string `env:"HOSTNAME"`
	Port     string `env:"PORT,notEmpty"`
}

type AccessControl struct {
	AllowOrigin string `env:"ACCESS_CONTROL_ALLOW_ORIGIN"`
}

type Header struct {
	RefIDHeaderKey string `env:"REF_ID_HEADER_KEY,notEmpty"`
}

type Database struct {
	Host     string `env:"DB_HOST,notEmpty"`
	Port     string `env:"DB_PORT,notEmpty"`
	User     string `env:"DB_USER,notEmpty"`
	Password string `env:"DB_PASSWORD,notEmpty"`
	DBName   string `env:"DB_NAME,notEmpty"`
}

type SlipOK struct {
	BaseURL  string `env:"SLIPOK_BASE_URL,notEmpty"`
	BranchID string `env:"SLIPOK_BRANCH_ID,notEmpty"`
	APIKey   string `env:"SLIPOK_API_KEY,notEmpty"`
}

type HomeAssistant struct {
	BaseURL  string `env:"HA_BASE_URL,notEmpty"`
	Token    string `env:"HA_TOKEN,notEmpty"`
	EntityID string `env:"HA_FEEDER_ENTITY_ID,notEmpty"`
}

var once sync.Once
var config Config

func prefix(e string) string {
	if e == "" {
		return ""
	}

	return fmt.Sprintf("%s_", e)
}

func C(envPrefix string) Config {
	once.Do(func() {
		opts := env.Options{
			Prefix: prefix(envPrefix),
		}

		var err error
		config, err = parseEnv[Config](opts)
		if err != nil {
			log.Fatal(err)
		}
	})

	return config
}

// TODO: read config from xxx.yaml file that contains ${ENV} variable e.g. serviceDLTUrl: ${SERVICE_CORE_DLT_ACCOUNT_URL}
