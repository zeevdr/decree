package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds separate connection pools for read and write operations.
type DB struct {
	WritePool *pgxpool.Pool
	ReadPool  *pgxpool.Pool
}

// NewDB creates connection pools for the given DSNs.
// If readDSN is empty, the write pool is used for reads.
func NewDB(ctx context.Context, writeDSN, readDSN string) (*DB, error) {
	writePool, err := pgxpool.New(ctx, writeDSN)
	if err != nil {
		return nil, fmt.Errorf("connect to write db: %w", err)
	}
	if err := writePool.Ping(ctx); err != nil {
		writePool.Close()
		return nil, fmt.Errorf("ping write db: %w", err)
	}

	readPool := writePool
	if readDSN != "" && readDSN != writeDSN {
		readPool, err = pgxpool.New(ctx, readDSN)
		if err != nil {
			writePool.Close()
			return nil, fmt.Errorf("connect to read db: %w", err)
		}
		if err := readPool.Ping(ctx); err != nil {
			writePool.Close()
			readPool.Close()
			return nil, fmt.Errorf("ping read db: %w", err)
		}
	}

	return &DB{
		WritePool: writePool,
		ReadPool:  readPool,
	}, nil
}

// Close closes both connection pools.
func (db *DB) Close() {
	if db.ReadPool != db.WritePool {
		db.ReadPool.Close()
	}
	db.WritePool.Close()
}
