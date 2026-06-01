package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/NP-compete/arcana/pkg/logger"
	_ "github.com/lib/pq"
)

var log = logger.New("db")

func Connect() (*sql.DB, error) {
	dsn := buildDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			log.Info("connected", "mode", dbMode(), "host", resolveHost())
			return db, nil
		}
		log.Info("waiting for postgres", "attempt", i+1, "max", 30)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("db: failed to connect after 30 retries: %w", err)
}

func MustConnect() *sql.DB {
	db, err := Connect()
	if err != nil {
		log.Error("fatal: could not connect to database", "error", err)
		os.Exit(1)
	}
	return db
}

func ConnectTo(host, port, user, password, dbname, sslmode string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			log.Info("connected", "host", host, "dbname", dbname)
			return db, nil
		}
		log.Info("waiting for postgres", "attempt", i+1, "max", 30, "host", host)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("db: failed to connect to %s after 30 retries: %w", host, err)
}

func buildDSN() string {
	host := resolveHost()
	port := resolvePort()
	user := envOr("POSTGRES_USER", "arcana")
	dbname := envOr("POSTGRES_DB", "arcana")

	var pass, sslmode string
	if os.Getenv("ARCANA_ENV") == "production" {
		pass = os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			log.Error("POSTGRES_PASSWORD required in production")
			os.Exit(1)
		}
		sslmode = envOr("POSTGRES_SSLMODE", "require")
	} else {
		pass = envOr("POSTGRES_PASSWORD", "arcana-dev")
		sslmode = envOr("POSTGRES_SSLMODE", "prefer")
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)
}

func dbMode() string {
	return envOr("DB_MODE", "local")
}

func resolveHost() string {
	if h := os.Getenv("POSTGRES_HOST"); h != "" {
		return h
	}

	switch dbMode() {
	case "cnpg":
		return envOr("CNPG_CLUSTER_HOST", "arcana-db-rw.arcana.svc.cluster.local")
	case "external":
		return envOr("EXTERNAL_DB_HOST", "localhost")
	default:
		return "postgres"
	}
}

func resolvePort() string {
	if p := os.Getenv("POSTGRES_PORT"); p != "" {
		return p
	}

	switch dbMode() {
	case "external":
		return envOr("EXTERNAL_DB_PORT", "5432")
	default:
		return "5432"
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
