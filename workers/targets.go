package workers

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"time"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/models"
)

func runTargetsWorker(ctx context.Context, db *sql.DB) {
	targets, err := models.LoadAllActiveTargets(models.QueryWithDatabase(ctx, db))
	if err != nil {
		logger.Fatalln(err)
	}

	cancelFuncs := make(map[string]context.CancelFunc)
	logger.Printf("starting monitoring for %d targets\n", len(targets))

	for _, target := range targets {
		targetCtx, cancel := context.WithCancel(ctx)
		cancelFuncs[target.Id()] = cancel

		go runTargetWorker(targetCtx, db, &target)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			newTargets, err := models.LoadAllActiveTargets(models.QueryWithDatabase(ctx, db))
			if err != nil {
				logger.Fatalln(err)
			}

			newTargetIds := make(map[string]bool)

			for _, target := range newTargets {
				newTargetIds[target.Id()] = true

				_, existingTarget := cancelFuncs[target.Id()]
				if existingTarget {
					continue
				}

				targetCtx, cancel := context.WithCancel(ctx)
				cancelFuncs[target.Id()] = cancel

				logger.Printf("starting monitoring for new target name=%s\n", target.DisplayName())
				go runTargetWorker(targetCtx, db, &target)
			}

			for targetId, cancelFunc := range cancelFuncs {
				stillExists := newTargetIds[targetId]
				if !stillExists {
					targetIdx := slices.IndexFunc(targets, func(t models.Target) bool {
						return t.Id() == targetId
					})
					target := targets[targetIdx]

					logger.Printf("stopping monitoring for deleted target %s\n", target.DisplayName())
					cancelFunc()
					delete(cancelFuncs, targetId)
				}
			}

			targets = newTargets
			time.Sleep(time.Second)
		}
	}
}

func runTargetWorker(ctx context.Context, db *sql.DB, target *models.Target) {
	var timer <-chan time.Time
	next := func() {
		timer = time.After(time.Duration(target.Period) * time.Second)
	}
	next()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer:
			req, err := http.NewRequestWithContext(ctx, target.Method, target.Uri, nil)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("%s: failed: %v\n", target.DisplayName(), err)

				next()
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Printf("%s: unhealthy: %v\n", target.DisplayName(), err)

				next()
				continue
			}

			if res.StatusCode != config.DefaultResponseCode {
				logger.Printf("%s: unhealthy status=%d\n", res.StatusCode)

				next()
				continue
			}

			next()
		}
	}
}
