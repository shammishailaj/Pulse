package config

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

//Configuration is the main configurations for the application
type Configuration struct {
	LogList    []string `toml:"LogList"`
	EmailList  []string `toml:"EmailList"`
	OutputFile string   `toml:"OutputFile"`
	SMTPConfig string   `toml:"SMTPConfig"`
	Port       int      `toml:"Port"`
}

//SMTPConfig is the configurations for a personal SMTP server a user would like to use
type SMTPConfig struct {
	Server Server
	User   User
}

//Server is the SMTP Server
type Server struct {
	Host string `toml:"Host"`
	Port int    `toml:"Port"`
}

//User is the User/Pass combination for the SMTP Server
type User struct {
	UserName string `toml:"UserName"`
	PassWord string `toml:"PassWord"`
}

//SecretConfig is the configurations to hold the keys for MailGun
type SecretConfig struct {
	Sender     string `toml:"Sender"`
	Domain     string `toml:"Domain"`
	PrivateKey string `toml:"PrivateKey"`
	PublicKey  string `toml:"PublicKey"`
}

var (
	mailGunConfig = "MailGun.toml"
	pulseConfig   = "PulseConfig.toml"
	smtpConfig    string
)

//Load returns the main configuration
func Load() (*Configuration, error) {

	cfg := &Configuration{}
	if _, err := toml.DecodeFile(pulseConfig, cfg); err != nil {
		home, err := homedir.Dir()
		if err != nil {
			return nil, fmt.Errorf("config.Load: Could not find %s in the executable directory and could not find home directory", pulseConfig)
		}
		if _, err := toml.DecodeFile(filepath.Join(home, pulseConfig), cfg); err != nil {
			return nil, fmt.Errorf("config.Load: Could not find %s in the %s or executable directory", pulseConfig, home)
		}
	}
	return cfg, nil
}

//LoadSMTP loads the settings for the smtp server
func LoadSMTP() (*SMTPConfig, error) {
	maincfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("config.LoadSMTP: %s", err)
	}
	cfg := &SMTPConfig{}
	if _, err := toml.DecodeFile(maincfg.SMTPConfig, cfg); err != nil {
		return nil, fmt.Errorf("config.LoadSMTP: %s", err)
	}
	return cfg, nil
}

//LoadSecret loads the keys for Mailgun
func LoadSecret() (*SecretConfig, error) {
	cfg := &SecretConfig{}
	if _, err := toml.DecodeFile(mailGunConfig, cfg); err != nil {
		return nil, fmt.Errorf("config.LoadSecret: %s", err)
	}
	return cfg, nil
}
