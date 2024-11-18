package workers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/models"
)

func runTargetsWorker(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, "SELECT * FROM targets")
	if err != nil {
		logger.Fatalln(err)
	}

	targets, err := models.LoadTargets(rows)
	if err != nil {
		logger.Fatalln(err)
	}

	for _, target := range targets {
		go func() {
			var timer <-chan time.Time
			next := func() {
				timer = time.After(time.Duration(target.Period) * time.Second)
			}
			next()

			logger.Printf("%s: running period=%d method=%s\n", target.DisplayName(), target.Period, target.Method)

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

					logger.Printf("%s: healthy\n", target.DisplayName())
					next()
				}
			}
		}()
	}
}
