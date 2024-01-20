package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"regexp"
	"time"
)

type User struct {
	ID           string
	Username     string
	Email        string
	CreatedAt    time.Time
	DiscordToken string
	TwitchToken  string
}

// GetUserByCredentials returns the matching user for the given credentials. If the credentials
// don't match an existing user GetUserByCredentials returns nil.
func GetUserByCredentials(username, password string) *User {
	if len(username) == 0 {
		return nil
	}

	sha256 := sha256.Sum256([]byte(password))
	password = hex.EncodeToString(sha256[:])

	if regexp.MustCompile("^\\w+$").MatchString(username) {
		return getUser("username=? AND password=?", username, password)
	} else {
		return getUser("email=? AND password=?", username, password)
	}

}

// GetUserByID returns the matching user for the given credentials. If the credentials
// don't match an existing user GetUserByID returns nil.
func GetUserByID(ID string) *User {
	return getUser("id=?", ID)
}

// getUser is a helper function to get a user from the database for the given condition
func getUser(where string, args ...any) *User {
	var (
		dbID        string
		dbUsername  string
		dbEmail     string
		dbCreatedAt time.Time
		dbDiscord   string
		dbTwitch    string
	)

	err := QueryRow(`SELECT id,username,email,create_time,discord,twitch
		FROM users
		WHERE `+where+`;`,
		args...).
		Scan(&dbID, &dbUsername, &dbEmail, &dbCreatedAt, &dbDiscord, &dbTwitch)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.Printf("Error getting user from database: %v", err)
		return nil
	}

	return &User{
		ID:           dbID,
		Username:     dbUsername,
		Email:        dbEmail,
		CreatedAt:    dbCreatedAt,
		DiscordToken: dbDiscord,
		TwitchToken:  dbTwitch,
	}
}
