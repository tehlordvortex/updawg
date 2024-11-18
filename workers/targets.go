package workers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/models"
	"github.com/tehlordvortex/updawg/pubsub"
)

type targetState struct {
	cancel context.CancelFunc
	target *models.Target
}

func runTargetsWorker(ctx context.Context, db *sql.DB) {
	targetsPubSub, unsub, err := pubsub.SubscribeMany(ctx, []string{models.TargetCreatedTopic, models.TargetDeletedTopic})
	if err != nil {
		logger.Fatalln(err)
	}
	defer unsub()

	targets, err := models.FindAllActiveTargets(ctx, db)
	if err != nil {
		logger.Fatalln(err)
	}

	states := make(map[string]targetState)
	logger.Printf("starting monitoring for %d targets\n", len(targets))

	for _, target := range targets {
		targetCtx, cancel := context.WithCancel(ctx)
		state := targetState{cancel: cancel, target: &target}
		states[target.Id()] = state

		go runTargetWorker(targetCtx, db, &state)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-targetsPubSub:
			switch message.Topic {
			case models.TargetCreatedTopic:
				_, exists := states[message.Msg]
				if exists {
					continue
				}

				target, err := models.FindTargetById(ctx, db, message.Msg)
				if err != nil {
					logger.Printf("failed to load target %s: %v\n", message.Msg, err)
					continue
				}

				targetCtx, cancel := context.WithCancel(ctx)
				state := targetState{cancel: cancel, target: &target}
				states[target.Id()] = state

				logger.Printf("starting monitoring for new target id=%s name=%s\n", target.Id(), target.DisplayName())
				go runTargetWorker(targetCtx, db, &state)
			case models.TargetDeletedTopic:
				state, exists := states[message.Msg]
				if !exists {
					continue
				}

				logger.Printf("stopping monitoring for deleted target id=%s\n", message.Msg)
				state.cancel()
				delete(states, message.Msg)
			}
		}
	}
}

func runTargetWorker(ctx context.Context, db *sql.DB, state *targetState) {
	defer state.cancel()

	targetPubSub, unsub, err := pubsub.Subscribe(ctx, models.TargetUpdatedTopic)
	if err != nil {
		logger.Fatalln("pubsub.Sub failed:", err)
	}
	defer unsub()

	var timer <-chan time.Time
	next := func() {
		timer = time.After(time.Duration(state.target.Period) * time.Second)
	}
	next()

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-targetPubSub:
			if message.Msg != state.target.Id() {
				continue
			}

			if err := state.target.Reload(ctx, db); err != nil {
				if err == sql.ErrNoRows {
					return
				}

				logger.Printf("failed to reload target: %v id=%", err, state.target.Id())
				return
			}
		case <-timer:
			req, err := http.NewRequestWithContext(ctx, state.target.Method, state.target.Uri, nil)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("health check failed id=%s name=%s error=%v\n", state.target.Id(), state.target.DisplayName(), err)

				next()
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("health check failed id=%s name=%s error=%v\n", state.target.Id(), state.target.DisplayName(), err)

				next()
				continue
			}

			if res.StatusCode != config.DefaultResponseCode {
				logger.Printf("health check failed id=%s name=%s status=%d\n", state.target.Id(), state.target.DisplayName(), res.StatusCode)

				next()
				continue
			}

			next()
		}
	}
}
