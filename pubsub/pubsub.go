package pubsub

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tehlordvortex/updawg/config"
)

var ErrNotRunning = fmt.Errorf("pubsub is not running")

const CreatePubsubTableSql = (`
CREATE TABLE IF NOT EXISTS pubsub (
	topic varchar(255) NOT NULL,
	message text NOT NULL,
	timestamp integer NOT NULL
);

CREATE INDEX IF NOT EXISTS pubsub_on_topic_and_timestamp ON pubsub (topic, timestamp);
CREATE INDEX IF NOT EXISTS pubsub_on_timestamp ON pubsub (timestamp);
	`)

type Message struct {
	Topic string
	Msg   string
	Ts    time.Time
}

var (
	db        *sql.DB
	mut       = &sync.Mutex{}
	topicSubs = make(map[string][]chan Message)
	logger    = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Lshortfile)
)

func Run(ctx context.Context) {
	var err error

	_db, err := sql.Open("sqlite", config.GetPubsubDatabaseUri())
	if err != nil {
		logger.Fatalln("failed to start pubsub:", err)
	}

	_, err = _db.ExecContext(ctx, CreatePubsubTableSql)
	if err != nil {
		logger.Fatalln("failed to start pubsub:", err)
	}

	var lastTsUnix int64
	err = _db.QueryRowContext(ctx, "SELECT timestamp FROM pubsub ORDER BY timestamp DESC LIMIT 1").Scan(&lastTsUnix)
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Fatalln("failed to start pubsub:", err)
		}
	}

	db = _db
	logger.Printf("pubsub streaming messages after ts=%d", lastTsUnix)

	go func() {
		getTopics := func() []string {
			mut.Lock()
			defer mut.Unlock()

			var topics []string
			for topic := range topicSubs {
				topics = append(topics, topic)
			}

			return topics
		}

		notify := func(msg Message) {
			mut.Lock()
			defer mut.Unlock()

			subs := topicSubs[msg.Topic]
			for _, ch := range subs {
				ch <- msg
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				messages, lastTs, err := loadMessagesForTopics(ctx, getTopics(), lastTsUnix)
				if err != nil {
					logger.Println(err)
					return
				}

				for _, message := range messages {
					logger.Printf("pubsub.Recv(%s): %s ts=%d", message.Topic, message.Msg, message.Ts.UnixMilli())
					notify(message)
				}

				if lastTs != 0 {
					lastTsUnix = lastTs
				}
				time.Sleep(time.Millisecond)
			}
		}
	}()
}

func Publish(ctx context.Context, topic string, message string) error {
	if db == nil {
		logger.Printf("pubsub.Publish(%s): failed: %v", topic, ErrNotRunning)
		return ErrNotRunning
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Printf("pubsub.Publish(%s): failed: %v", topic, err)
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	logger.Printf("pubsub.Publish(%s): %s ts=%d", topic, message, now.UnixMilli())
	_, err = tx.ExecContext(ctx, "INSERT INTO pubsub (topic, message, timestamp) VALUES (?, ?, ?)", topic, message, now.UnixMilli())
	if err != nil {
		logger.Printf("pubsub.Publish(%s): failed: %v", topic, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.Printf("pubsub.Publish(%s): failed: %v", topic, err)
		return err
	}

	return nil
}

func Subscribe(ctx context.Context, topic string) (<-chan Message, context.CancelFunc, error) {
	return SubscribeMany(ctx, []string{topic})
}

func SubscribeMany(ctx context.Context, topics []string) (<-chan Message, context.CancelFunc, error) {
	if db == nil {
		return nil, nil, ErrNotRunning
	}

	ch := make(chan Message)
	unsub := func() {
		mut.Lock()
		defer mut.Unlock()

		for _, topic := range topics {
			subs := topicSubs[topic]
			subs = slices.DeleteFunc(subs, func(c chan Message) bool {
				return c == ch
			})
			topicSubs[topic] = subs
		}
	}

	mut.Lock()
	defer mut.Unlock()

	for _, topic := range topics {
		subs := topicSubs[topic]
		subs = append(subs, ch)
		topicSubs[topic] = subs
	}

	return ch, unsub, nil
}

func loadMessagesForTopics(ctx context.Context, topics []string, timestamp int64) ([]Message, int64, error) {
	newTopics := make([]string, len(topics))
	for i, topic := range topics {
		newTopics[i] = "'" + topic + "'"
	}
	topicsSql := strings.Join(newTopics, ", ")

	query := fmt.Sprintf("SELECT topic, message, timestamp FROM pubsub WHERE topic IN (%s) AND timestamp > %d", topicsSql, timestamp)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("loadMessagesForTopics(%s): %v", topicsSql, err)
	}
	defer rows.Close()

	var messages []Message
	var lastTimestamp int64

	for rows.Next() {
		var topic, message string
		var timestampUnix int64

		if err := rows.Scan(&topic, &message, &timestampUnix); err != nil {
			return nil, 0, fmt.Errorf("loadMessagesForTopics(%s): %v", topicsSql, err)
		}

		messages = append(messages, Message{topic, message, time.UnixMilli(timestampUnix)})
		lastTimestamp = timestampUnix
	}

	return messages, lastTimestamp, nil
}
