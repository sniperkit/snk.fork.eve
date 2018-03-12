package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	// driver for sqlite
	_ "github.com/mattn/go-sqlite3"
)

// default databaseConnection to use
var dbConection struct {
	// Path is the path of the sqlite file
	Path string

	// connection to database driver
	db *sql.DB
}

// User holds user data
type User struct {

	// UserID in database
	ID int

	// Username
	Name string
}

// Connect creates a connection to database
func Connect(path string) error {
	dbConection.Path = path
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	dbConection.db = db
	return nil
}

// Close closes a connection to database
func Close() error {
	// check if database is conntected
	if dbConection.db == nil {
		return ErrConnectionClosed
	}
	defer func() { dbConection.db = nil }()
	return dbConection.db.Close()
}

// CreateUser adds user to database
func CreateUser(userName, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("cannot hash password: " + err.Error())
	}
	_, err = dbConection.db.Exec("INSERT INTO User(Name,PasswordHash) VALUES($1,$2)", userName, hash)
	return err
}

// CheckCredentials checks if combination of userName and password is valid
// User object data is returned if credentials are valid
func CheckCredentials(userName, password string) (*User, error) {
	var hash []byte
	user := User{
		Name: userName,
	}
	var sessionKey sql.NullString
	err := dbConection.db.QueryRow("SELECT UserID,PasswordHash,SessionKey FROM User WHERE Name=$1", user.Name).Scan(&user.ID, &hash, &sessionKey)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotExists
	} else if err != nil {
		return nil, ErrInternalServerError
	}
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return nil, ErrWrongPassword
	} else if err != nil {
		return nil, ErrInternalServerError
	}
	return &user, nil
}

//StoreSessionKey saves the key in the database for the given user and saves the key to the user struct
func StoreSessionKey(user *User, key string) bool {
	result, err := dbConection.db.Exec("UPDATE User SET SessionKey=$1 WHERE UserId = $2", key, user.ID)
	if err != nil {
		log.Println("cannot update session key for user ", user.ID, err)
		return false
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("session key was not stored:", err)
		return false
	} else if rows != 1 {
		log.Println("cannot update session key for user: ", user.ID)
		return false
	}
	return true
}

//GetUserForSession gets the user associated with the given session key
func GetUserForSession(sessionKey string) *User {
	var user User
	err := dbConection.db.QueryRow("SELECT UserID,Name FROM User WHERE SessionKey=$1", sessionKey).Scan(&user.ID, &user.Name)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("cannot get user for sessionID:", err)
		}
		return nil
	}
	return &user
}

// Bot represents database entry of a bot
type Bot struct {
	ID        int
	Name      string
	Image     string
	Gender    string
	User      int
	Affection float64
	Mood      float64
}

// CreateBot creates a bot entry in the database and fills the empty values in the given bot struct
func CreateBot(bot *Bot) error {
	v, err := dbConection.db.Exec("INSERT INTO Bot(Name,Image,Gender,User) VALUES($1,$2,$3,$4)", bot.Name, bot.Image, bot.Gender, bot.User)
	if err != nil {
		return err
	}
	botID, err := v.LastInsertId()
	if err != nil {
		return err
	}
	bot.ID = int(botID)
	return nil
}

// MessageSender defines who send a message
type MessageSender int

const (
	// BotIsSender says that the message was sent by the bot
	BotIsSender = 0

	// UserIsSender says that the message was sent by the user
	UserIsSender = 1
)

// Message represents database entry of a message
type Message struct {
	ID        int
	Bot       int
	Sender    MessageSender
	Timestamp time.Time
	Content   string
	Rating    float64
}

// MessageMaxLength defines the maximum message length
const MessageMaxLength = 200

// StoreMessage saves message in database
func StoreMessage(userID int, msg Message) error {
	if len(msg.Content) > MessageMaxLength {
		return errors.New("message to long")
	} else if len(msg.Content) < 1 {
		return errors.New("cannot store empty message")
	}
	exists, err := rowExists("SELECT * FROM Bot WHERE BotID=$1 AND User=$2", msg.Bot, userID)
	if err != nil {
		return err
	} else if !exists {
		return errors.New("bot does not belong to user")
	}
	_, err = dbConection.db.Exec("INSERT INTO Message(Bot,Sender,Timestamp,Content) VALUES($1,$2,$3,$4)", msg.Bot, int(msg.Sender), msg.Timestamp, msg.Content)
	return err
}

// GetMessages returns a list of all messages, that the user and bot sent each other
func GetMessages(user, bot int) (*[]Message, error) {
	rows, err := dbConection.db.Query(`
		SELECT 	Timestamp,
				Content,
				Rating 
		FROM Message 
		WHERE Bot=$1 AND User=$2`, bot, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []Message
	var cursor Message
	for rows.Next() {
		if err := rows.Scan(&cursor.Timestamp, &cursor.Content, &cursor.Rating); err == nil {
			messages = append(messages, cursor)
		} else {
			log.Println(err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &messages, nil
}

// GetBotsForUser returns all bots which belong to the given user
func GetBotsForUser(userID int) (*[]Bot, error) {
	rows, err := dbConection.db.Query(`
		SELECT 	BotID,
				Name,
				Image,
				Gender,
				Affection, 
				Mood
		FROM Bot 
		WHERE User=$2`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	var cursor Bot
	for rows.Next() {
		if err := rows.Scan(&cursor.ID, &cursor.Name, &cursor.Image, &cursor.Gender, &cursor.Affection, &cursor.Mood); err == nil {
			bots = append(bots, cursor)
		} else {
			log.Println(err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &bots, nil
}

func rowExists(query string, args ...interface{}) (bool, error) {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := dbConection.db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}