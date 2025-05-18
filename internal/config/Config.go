package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", home, configFileName))
	if err != nil {
		return err
	}
	defer file.Close()

	data, _ := json.Marshal(c)
	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func Read() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(fmt.Sprintf("%s/%s", home, configFileName))
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	config := Config{}
	data, err := io.ReadAll(file)
	if err != nil {
		return Config{}, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
