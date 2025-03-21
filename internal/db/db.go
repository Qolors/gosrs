package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type DBInstance struct {
	Conn *pgxpool.Pool
}

func InitializeConnection(ctx context.Context) (*DBInstance, error) {

	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	connStr := os.Getenv("DATABASE_URL")

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	// Recommended pool settings (adjust as needed)
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour       // recycle connections regularly
	config.MaxConnIdleTime = 5 * time.Minute // prevent idle connections

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &DBInstance{Conn: pool}, err
}
