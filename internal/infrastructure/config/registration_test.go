package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestRegistrationDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Registration.Enabled || cfg.Registration.Address() != "127.0.0.1:8081" {
		t.Fatal("unsafe registration defaults")
	}
}

func TestRegistrationEnvironment(t *testing.T) {
	t.Setenv("MCGAME_REGISTRATION_ENABLED", "true")
	t.Setenv("MCGAME_REGISTRATION_HOST", "::1")
	t.Setenv("MCGAME_REGISTRATION_PORT", "8181")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Registration.Enabled || cfg.Registration.Address() != "[::1]:8181" {
		t.Fatal("registration env override not applied")
	}
}
