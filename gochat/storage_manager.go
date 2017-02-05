package gochat

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/Sirupsen/logrus"
	"database/sql/driver"
)

type StorageManager struct {
	config DatabaseConfig
	logger *log.Entry
	db     *sqlx.DB
}

func NewStorageManager(config DatabaseConfig, logger *log.Entry) (*StorageManager, error) {
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
			connection_string += " password=" + config.Password
		}

		if config.Host != "" {
			connection_string += " host=" + config.Host
		}

		if config.Port != "" {
			connection_string += " port=" + config.Port
		}

		db, err = sqlx.Open("postgres", connection_string)
	default:
		return &StorageManager{}, errors.New("Unable to determine DB type")
	}

	if err != nil {
		return &StorageManager{}, err
	}

	return &StorageManager{config: config, logger: logger, db: db}, nil
}

func (manager *StorageManager) CloseStorage() error {
	return manager.db.Close()
}

// Convenience wrapper around the DB's Exec call result, it returns any errors Exec encounters
// It also has an affectedCheck anon function that ensure the correct number of rows is affected
func (manager *StorageManager) CheckExecOutcome(result driver.Result, err error, affectedCheck func(int64) bool) error {
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if !affectedCheck(affected) {
		return errors.New("We affected a different number of rows than we were expecting (" + string(affected) + ")")
	}

	return nil
}

func (manager *StorageManager) ExecZeroOrMoreRows(result driver.Result, err error) error {
	return manager.CheckExecOutcome(result, err, func(affected int64) bool { return affected >= 0 })
}

func (manager *StorageManager) ExecOneRow(result driver.Result, err error) error {
	return manager.CheckExecOutcome(result, err, func(affected int64) bool { return affected == 1 })
}

func (manager *StorageManager) ExecAtLeastOneRow(result driver.Result, err error) error {
	return manager.CheckExecOutcome(result, err, func(affected int64) bool { return affected > 0 })
}
