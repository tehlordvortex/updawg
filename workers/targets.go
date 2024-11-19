package workers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/models"
	"github.com/tehlordvortex/updawg/pubsub"
)

type targetState struct {
	cancel context.CancelFunc
	target *models.Target
}

func runTargetsWorker(ctx context.Context, rwdb *database.RWDB, ps *pubsub.PubSub) {
	targetsPubSub, unsub, err := ps.SubscribeMany(ctx, []string{models.TargetCreatedTopic, models.TargetDeletedTopic}, 1)
	if err != nil {
		logger.Fatalln(err)
	}
	defer unsub()

	targets, err := models.FindAllActiveTargets(ctx, rwdb.Read())
	if err != nil {
		logger.Fatalln(err)
	}

	cancelFuncs := make(map[string]context.CancelFunc)
	logger.Printf("starting monitoring for %d targets\n", len(targets))

	for _, target := range targets {
		targetCtx, cancel := context.WithCancel(ctx)
		cancelFuncs[target.Id()] = cancel

		go runTargetWorker(targetCtx, rwdb, ps, target)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-targetsPubSub:
			switch message.Topic {
			case models.TargetCreatedTopic:
				_, exists := cancelFuncs[message.Msg]
				if exists {
					continue
				}

				target, err := models.FindTargetById(ctx, rwdb.Read(), message.Msg)
				if err != nil {
					logger.Printf("failed to load target %s: %v\n", message.Msg, err)
					continue
				}

				targetCtx, cancel := context.WithCancel(ctx)
				cancelFuncs[target.Id()] = cancel

				logger.Printf("starting monitoring for new target id=%s name=%s\n", target.Id(), target.DisplayName())
				go runTargetWorker(targetCtx, rwdb, ps, target)
			case models.TargetDeletedTopic:
				cancel, exists := cancelFuncs[message.Msg]
				if !exists {
					continue
				}

				logger.Printf("stopping monitoring for deleted target id=%s\n", message.Msg)
				cancel()
				delete(cancelFuncs, message.Msg)
			}
		}
	}
}

func runTargetWorker(ctx context.Context, rwdb *database.RWDB, ps *pubsub.PubSub, target *models.Target) {
	targetPubSub, unsub, err := ps.Subscribe(ctx, models.TargetUpdatedTopic, 1)
	if err != nil {
		logger.Fatalln("pubsub.Sub failed:", err)
	}
	defer unsub()

	var timer <-chan time.Time
	next := func() {
		timer = time.After(time.Duration(target.Period) * time.Second)
	}
	next()

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-targetPubSub:
			if message.Msg != target.Id() {
				continue
			}

			if err := target.Reload(ctx, rwdb.Read()); err != nil {
				if err == sql.ErrNoRows {
					return
				}

				logger.Printf("failed to reload target: %v id=%", err, target.Id())
				return
			}
		case <-timer:
			req, err := http.NewRequestWithContext(ctx, target.Method, target.Uri, nil)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("health check failed id=%s name=%s error=%v\n", target.Id(), target.DisplayName(), err)

				next()
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("health check failed id=%s name=%s error=%v\n", target.Id(), target.DisplayName(), err)

				next()
				continue
			}

			if res.StatusCode != config.DefaultResponseCode {
				logger.Printf("health check failed id=%s name=%s status=%d\n", target.Id(), target.DisplayName(), res.StatusCode)

				next()
				continue
			}

			next()
		}
	}
}
