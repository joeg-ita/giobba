package config

import (
	"errors"
	"log"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

const (
	Development        = "dev"
	Production         = "prod"
	ConfigBaseFileName = "giobba"
)

type Database struct {
	Url             string `yaml:"url" env:"GIOBBA_DATABASE_URL"`
	Port            string `yaml:"port" env:"GIOBBA_DATABASE_PORT"`
	Username        string `yaml:"username" env:"GIOBBA_DATABASE_ADMIN_USERNAME"`
	Password        string `yaml:"password" env:"GIOBBA_DATABASE_ADMIN_PASSWORD"`
	DB              string `yaml:"db" env:"GIOBBA_DATABASE"`
	TasksCollection string `yaml:"tasksCollection"`
	JobsCollection  string `yaml:"jobsCollection"`
}

type Broker struct {
	Url      string `yaml:"url" env:"GIOBBA_BROKER_URL"`
	Port     string `yaml:"port" env:"GIOBBA_BROKER_PORT"`
	Username string `yaml:"username" env:"GIOBBA_BROKER_ADMIN_USERNAME"`
	Password string `yaml:"password" env:"GIOBBA_BROKER_ADMIN_PASSWORD"`
	DB       string `yaml:"db" env:"GIOBBA_BROKER_DB"`
}

type Config struct {
	Name               string   `yaml:"name"`
	Version            string   `yaml:"version"`
	Database           Database `yaml:"database"`
	Broker             Broker   `yaml:"broker"`
	Queues             []string `yaml:"queues"`
	WorkersNumber      int      `yaml:"workersNumber"`
	LockDuration       int      `yaml:"lockDuration"`
	JobsTimeoutRefresh int      `yaml:"jobsTimeoutRefresh"`
	PollingTimeout     int      `yaml:"pollingTimeout"`
}

func LoadConfig() (*Config, error) {

	// Determine environment
	env := os.Getenv("GIOBBA_ENV")
	if env != "" {
		env = "." + env
	}

	configFileExt := []string{".yml", ".yaml"}

	homePath, ok := os.LookupEnv("HOME")
	if ok {
		homePath += "/.giobba"
	}

	pwdPath, _ := os.LookupEnv("PWD")

	// Add config paths
	configPaths := []string{
		"/etc/giobba.d/",
		homePath + "/",
		pwdPath + "/",
	}

	var err error
	var config *Config
	for i := range len(configPaths) {

		for _, ext := range configFileExt {
			path := configPaths[i] + ConfigBaseFileName + env + ext

			_, err = os.Stat(path)
			if err == nil {
				config, err = ConfigFromYaml(path)
				if err != nil {
					err = errors.New("something went wrong")
				} else {
					break
				}
				break
			}
		}

	}

	return config, err
}

func ConfigFromYaml(filePath string) (*Config, error) {

	// Open the YAML file
	log.Printf("ConfigFromYaml - Loading file: %v", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("error loading file %v: %v", filePath, err.Error())
		return nil, err
	}
	defer file.Close()

	// Decode the YAML file into the struct
	decoder := yaml.NewDecoder(file)

	var config = &Config{}
	err = decoder.Decode(config)
	if err != nil {
		log.Printf("ConfigFromYaml - Error Decoding: %v", err)
	}

	// Print the loaded data
	err = OverrideEnvs(config)
	if err != nil {
		return nil, err
	}
	log.Printf("ConfigFromYaml - Config Loaded!", config)

	return config, nil
}

func OverrideEnvs(obj interface{}) error {
	return setFieldFromEnv(reflect.ValueOf(obj))
}

func setFieldFromEnv(v reflect.Value) error {

	t := v.Type()

	// If it's a pointer, get the underlying element
	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	// If it's not a struct, we're done
	if t.Kind() != reflect.Struct {
		return nil
	}

	// Iterate through all fields
	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// If the field is a struct, recursively process it
		if field.Type.Kind() == reflect.Struct {
			if err := setFieldFromEnv(fieldValue); err != nil {
				return err
			}
			continue
		}

		// Check if field has env tag
		checkEnv(field, fieldValue)
	}

	return nil
}

func checkEnv(field reflect.StructField, fieldValue reflect.Value) {
	// Get environment variable value
	envTag := field.Tag.Get("env")
	if envTag != "" {
		envValue := os.Getenv(envTag)
		if envValue != "" {
			// Set the field value
			if fieldValue.CanSet() {
				fieldValue.SetString(envValue)
			}
		}
	}
}
