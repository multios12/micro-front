package config

import "os"

func Load() Config {
	dataDir := getenv("DATA_DIR", "./data")
	return Config{
		Addr:            getenv("ADDR", ":3001"),
		AdminStaticDir:  getenv("ADMIN_STATIC_DIR", "web/static"),
		PublicStaticDir: getenv("STATIC_EXPORT_DIR", "./data/publish"),
		DataDir:         dataDir,
		DBPath:          getenv("DB_PATH", dataDir+"/app.db"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
