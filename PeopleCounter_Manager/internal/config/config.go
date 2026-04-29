package config

import (
	"PeopleCounter_Manager/internal/models"
	"encoding/json"
	"log"
	"os"
)

func LoadConfig(filename string) *models.Config {
	cfg := &models.Config{
		APIPort:      9000,
		JWTSecret:    "super_secret_jwt_key_please_change",
		DBConnString: "postgres://user:pass@localhost:5432/people_counter?sslmode=disable",
		KeepLogDays:  60,
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			data, _ := json.MarshalIndent(cfg, "", "  ")
			os.WriteFile(filename, data, 0644)
			return cfg
		}
		log.Fatalf("Ошибка открытия конфига: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(cfg); err != nil {
		log.Fatalf("Ошибка парсинга конфига: %v", err)
	}
	return cfg
}
