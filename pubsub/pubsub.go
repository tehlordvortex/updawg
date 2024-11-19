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

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
)

var ErrNotRunning = fmt.Errorf("pubsub is not running")

const CreatePubsubTableSql = (`
CREATE TABLE IF NOT EXISTS pubsub (
	id integer NOT NULL PRIMARY KEY,
	topic varchar(255) NOT NULL,
	message text NOT NULL,
	timestamp integer NOT NULL
);

-- CREATE INDEX IF NOT EXISTS pubsub_topic ON pubsub (topic);
-- CREATE INDEX IF NOT EXISTS pubsub_timestamp ON pubsub (timestamp);
-- CREATE INDEX IF NOT EXISTS pubsub_topic_timestamp ON pubsub (topic, timestamp);
-- CREATE INDEX IF NOT EXISTS pubsub_id_topic_timestamp ON pubsub (id, topic, timestamp);
	`)

type Message struct {
	Topic string
	Msg   string
	Ts    time.Time
}

type PublishMessage struct {
	ts       time.Time
	data     [][]string
	respChan chan error
}

type subscription struct {
	topics []string
	ch     chan *Message
}

type PubSub struct {
	rwdb      *database.RWDB
	mut       sync.Mutex
	topicSubs map[string][]subscription
	batchChan chan []any
	batchCtx  context.Context
}

const (
	Interval  = time.Millisecond * 1
	MaxVarNum = 32766
	ChunkSize = MaxVarNum / 3
)

var MaxQueueLen = ChunkSize

var logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Lshortfile)

func Connect(ctx context.Context, path string) (*PubSub, error) {
	fail := func(err error) error {
		return fmt.Errorf("pubsub.Connect(%s): %v", path, err)
	}

	var err error
	ps := &PubSub{mut: sync.Mutex{}, topicSubs: make(map[string][]subscription), batchChan: make(chan []any, MaxQueueLen)}

	ps.rwdb, err = database.Connect(context.Background(), path)
	if err != nil {
		return nil, fail(err)
	}

	_, err = ps.rwdb.Write().ExecContext(ctx, CreatePubsubTableSql)
	if err != nil {
		return nil, fail(err)
	}

	var lastId int64
	err = ps.rwdb.Write().QueryRowContext(ctx, "SELECT id FROM pubsub ORDER BY id DESC LIMIT 1").Scan(&lastId)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fail(err)
		}
	}
	logger.Printf("connnected after=%d", lastId)

	go ps.runListener(ctx, lastId)

	batchCtx, cancel := context.WithCancel(context.Background())
	ps.batchCtx = batchCtx
	go ps.runBatcher(ctx, batchCtx, cancel)

	return ps, nil
}

func (ps *PubSub) Publish(ctx context.Context, topic string, message string) error {
	return ps.PublishMany(ctx, [][]string{{topic, message}})
}

func (ps *PubSub) PublishAsync(topic, message string) {
	ps.batchChan <- []any{topic, message, time.Now().UnixMicro()}
}

func (ps *PubSub) PublishMany(ctx context.Context, topicsAndMessages [][]string) error {
	now := time.Now()

	for topicsAndMessages := range slices.Chunk(topicsAndMessages, ChunkSize) {
		var msgs [][]any

		for _, topicAndMessage := range topicsAndMessages {
			msgs = append(msgs, []any{topicAndMessage[0], topicAndMessage[1], now})
		}

		if err := ps.publishMessages(ctx, msgs); err != nil {
			return err
		}
	}

	return nil
}

func (ps *PubSub) PublishManyToTopic(ctx context.Context, topic string, messages []string) error {
	topicsAndMessages := make([][]string, 0, len(messages))
	for _, message := range messages {
		topicsAndMessages = append(topicsAndMessages, []string{topic, message})
	}

	return ps.PublishMany(ctx, topicsAndMessages)
}

func (ps *PubSub) Subscribe(ctx context.Context, topic string, buffer int) (<-chan *Message, context.CancelFunc, error) {
	return ps.SubscribeMany(ctx, []string{topic}, buffer)
}

func (ps *PubSub) SubscribeMany(ctx context.Context, topics []string, buffer int) (<-chan *Message, context.CancelFunc, error) {
	if ps.rwdb == nil {
		return nil, nil, ErrNotRunning
	}

	ch := make(chan *Message, buffer)
	sub := subscription{topics, ch}

	ps.mut.Lock()
	defer ps.mut.Unlock()

	for _, topic := range sub.topics {
		subs := ps.topicSubs[topic]
		subs = append(subs, sub)
		ps.topicSubs[topic] = subs
	}

	return ch, func() {
		ps.mut.Lock()
		defer ps.mut.Unlock()

		for _, topic := range sub.topics {
			subs := ps.topicSubs[topic]
			subs = slices.DeleteFunc(subs, func(s subscription) bool {
				return s.ch == sub.ch
			})
			ps.topicSubs[topic] = subs
		}
	}, nil
}

func (ps *PubSub) Wait() {
	<-ps.batchCtx.Done()
}

func (ps *PubSub) runBatcher(appCtx context.Context, batchCtx context.Context, cancel context.CancelFunc) {
	publishQueue := make([][]any, 0, MaxQueueLen)

	flushTicker := time.Tick(Interval)
	defer cancel()

	flush := func() {
		if err := ps.publishMessages(batchCtx, publishQueue); err != nil {
			logger.Println(err)
		}

		publishQueue = make([][]any, 0, MaxQueueLen)
	}

	for {
		select {
		case <-appCtx.Done():
			go func() {
				ps.batchChan <- nil
			}()
		case msg := <-ps.batchChan:
			if msg == nil {
				flush()
				return
			}

			publishQueue = append(publishQueue, msg)

			if len(publishQueue) >= MaxQueueLen {
				flush()
			}
		case <-flushTicker:
			flush()
		}
	}
}

func (ps *PubSub) publishMessages(ctx context.Context, msgs [][]any) error {
	mCount := len(msgs)
	if mCount == 0 {
		return nil
	}

	fail := func(err error) error {
		err = fmt.Errorf("publishMessages: failed: %v", err)
		return err
	}

	tx, err := ps.rwdb.Write().BeginTx(ctx, nil)
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback()

	var queryBuilder strings.Builder
	count := len(msgs)
	vars := make([]any, 0, 3*mCount)

	queryBuilder.WriteString("INSERT INTO pubsub (topic, message, timestamp) VALUES ")

	for i, publishMsg := range msgs {
		vars = append(vars, publishMsg...)

		queryBuilder.WriteString("(?, ?, ?)")
		if i < count-1 {
			queryBuilder.WriteString(", ")
		}
	}

	query := queryBuilder.String()
	_, err = tx.ExecContext(ctx, query, vars...)
	if err != nil {
		return fail(err)
	}

	if err := tx.Commit(); err != nil {
		return fail(err)
	}

	return nil
}

func (ps *PubSub) runListener(ctx context.Context, lastId int64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			rows, err := loadMessagesForTopics(ctx, ps.rwdb.Read(), ps.getTopics(), lastId)
			if err != nil {
				if err != context.Canceled {
					logger.Println(err)
				}

				return
			}

			var nRows int
			for rows.Next() {
				var topic, message string
				var id, timestampUnix int64

				if err := rows.Scan(&id, &topic, &message, &timestampUnix); err != nil {
					if err != context.Canceled {
						logger.Println(err)
					}

					rows.Close()
					return
				}

				ps.notify(topic, message, timestampUnix)

				lastId = id
				nRows += 1
			}

			rows.Close()
			time.Sleep(Interval)
		}
	}
}

func (ps *PubSub) getTopics() []string {
	ps.mut.Lock()
	defer ps.mut.Unlock()

	var topics []string
	for topic := range ps.topicSubs {
		topics = append(topics, topic)
	}

	return topics
}

func (ps *PubSub) notify(topic, message string, timestampUnix int64) {
	ps.mut.Lock()
	defer ps.mut.Unlock()

	subs := ps.topicSubs[topic]
	msg := &Message{topic, message, time.UnixMicro(timestampUnix)}

	for _, sub := range subs {
		// Ensure a slow receiver cannot block us
		go func() {
			sub.ch <- msg
		}()
	}
}

func loadMessagesForTopics(ctx context.Context, db *sql.DB, topics []string, id int64) (*sql.Rows, error) {
	fail := func(err error) (*sql.Rows, error) {
		if err == context.Canceled {
			return nil, err
		}

		return nil, fmt.Errorf("loadMessagesForTopics(%s): %v", strings.Join(topics, ", "), err)
	}

	var vars []any
	vars = append(vars, id)

	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, topic, message, timestamp FROM pubsub WHERE id > ? AND topic IN (")

	nTopics := len(topics)
	for i, topic := range topics {
		queryBuilder.WriteString("?")
		if i < nTopics-1 {
			queryBuilder.WriteString(", ")
		}

		vars = append(vars, topic)
	}

	queryBuilder.WriteString(") ORDER BY timestamp")
	query := queryBuilder.String()

	// fmt.Println(query, vars)
	rows, err := db.QueryContext(ctx, query, vars...)
	if err != nil {
		return fail(err)
	}

	return rows, nil
}
