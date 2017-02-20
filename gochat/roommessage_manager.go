package gochat

import (
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	CREATE_MESSAGE_SQL       = "INSERT INTO messages (user_id, room_id, message, epoch_timestamp) VALUES (?, ?, ?, ?)"
	GET_LATEST_ROOM_MESSAGES = `
	SELECT
		u.username AS username,
		r.name AS room_name,
		m.message AS message,
		m.epoch_timestamp AS epoch_timestamp
	FROM
		messages AS m
	JOIN
		users AS u ON (m.user_id = u.id)
		rooms AS r ON (m.room_id = r.id)
	WHERE
		m.room_id=?
		AND m.epoch_timestamp>=?
	`
	MESSAGE_SCHEMA = `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		room_id INTEGER,
		message TEXT,
		epoch_timestamp INT
	)`
)

type RoomMessageManager struct {
	storage *StorageManager
	logger  *log.Entry
}

func NewRoomMessageManager(storage *StorageManager, logger *log.Entry) (*RoomMessageManager, error) {
	// Create the messages table if it doesn't already exist
	_, err := storage.db.Exec(MESSAGE_SCHEMA)
	if err != nil {
		logger.Error(err)
		return &RoomMessageManager{}, errors.New("Failed to generate the TextMessage schema.")
	}

	manager := RoomMessageManager{
		storage: storage,
		logger:  logger,
	}

	return &manager, nil
}

func (manager *RoomMessageManager) PersistRoomMessage(user *ServerUser, room *ServerRoom, message string) error {
	sql := manager.storage.db.Rebind(CREATE_MESSAGE_SQL)
	err := manager.storage.ExecOneRow(manager.storage.db.Exec(sql, user.User.Id, room.Room.Id, message, time.Now().Unix()))
	if err != nil {
		manager.logger.Error(err)
		return errors.New("Failed to run CREATE_MESSAGE_SQL")
	}

	return nil
}

func (manager *RoomMessageManager) GetRoomMessagesSince(room *ServerRoom, timeSince time.Time) ([]*ServerRoomMessage, error) {
	var messages []*ServerRoomMessage
	sql := manager.storage.db.Rebind(GET_LATEST_ROOM_MESSAGES)
	rows, err := manager.storage.db.Queryx(sql, room.Room.Id, timeSince.Unix())
	if err != nil {
		manager.logger.Error(err)
		return messages, errors.New("Failed to run GET_LATEST_ROOM_MESSAGES")
	}

	for rows.Next() {
		var dbRoomMessage RoomMessage
		err = rows.StructScan(dbRoomMessage)
		if err != nil {
			manager.logger.Error(err)
			return messages, errors.New("Failed to parse GET_LATEST_ROOM_MESSAGES result into struct")
		}

		roomMessage := ServerRoomMessage{RoomMessage: &dbRoomMessage, Timestamp: time.Unix(dbRoomMessage.Timestamp, 0)}
		messages = append(messages, &roomMessage)
	}

	return messages, nil
}
