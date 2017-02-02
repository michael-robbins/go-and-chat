package gochat

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type StorageManager struct {
	config		DatabaseConfig
	db			*sqlx.DB
}

func NewStorageManager(config DatabaseConfig) (*StorageManager, error) {
	var db *sqlx.DB
	var err error

	switch config.Product {
	case "sqlite":
		db, err = sqlx.Open("sqlite3", config.Database)
	case "postgresql":
		connection_string := fmt.Sprint(
			"dbname="+config.Database,
			"user="+config.User,
		)

		if config.Password != "" {
			connection_string += " password="+config.Password
		}

		if config.Host != "" {
			connection_string += " host="+config.Host
		}

		if config.Port != "" {
			connection_string += " port="+config.Port
		}

		db, err = sqlx.Open("postgres", connection_string)
	default:
		return &StorageManager{}, errors.New("Unable to determine DB type")
	}

	if err != nil {
		return &StorageManager{}, err
	}

	return &StorageManager{config: config, db: db}, nil
}

func (manager *StorageManager) CloseStorage() error {
	return manager.db.Close()
}

// Convenience wrapper around the Exec call, it returns any errors Exec encounters
// It also has an affectedCheck anon function that ensure the correct number of rows is affected
func (manager *StorageManager) Exec(sql string, affectedCheck func(int64) bool, args ...interface{}) error {
	result, err := manager.db.Exec(DELETE_USER_SQL, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if !affectedCheck(affected) {
		return errors.New("We did not delete the user? We affected " + string(affected) + " rows")
	}

	return nil
}

func (manager *StorageManager) ExecOneRow(sql string, args ...interface{}) error {
	return manager.Exec(sql, func(affected int64) bool {return affected == 1}, args)
}

func (manager *StorageManager) ExecAtLeastOneRow(sql string, args ...interface{}) error {
	return manager.Exec(sql, func(affected int64) bool {return affected > 0}, args)
}
