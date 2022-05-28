package db

import (
	"fmt"
	"log"

	"github.com/tardisx/linkwallet/entity"
	bolthold "github.com/timshannon/bolthold"
)

type ConfigManager struct {
	db *DB
}

func NewConfigManager(db *DB) *ConfigManager {
	return &ConfigManager{db: db}
}

func (cmm *ConfigManager) LoadConfig() (entity.Config, error) {
	config := entity.Config{}
	err := cmm.db.store.FindOne(&config, &bolthold.Query{})
	if err == nil {
		if config.Version == 1 {
			return config, nil
		} else {
			return entity.Config{}, fmt.Errorf("failed to load config - wrong version %d", config.Version)
		}
	} else if err == bolthold.ErrNotFound {
		log.Printf("using default config")
		return cmm.DefaultConfig(), nil
	} else {
		return entity.Config{}, fmt.Errorf("failed to load config: %w", err)
	}
}

func (cmm *ConfigManager) DefaultConfig() entity.Config {
	return entity.Config{
		BaseURL: "http://localhost:8080",
		Version: 1,
	}
}

func (cmm *ConfigManager) SaveConfig(conf *entity.Config) error {
	err := cmm.db.store.Upsert("config", conf)
	if err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}
	return nil
}
