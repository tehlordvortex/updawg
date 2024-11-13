package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	gs := NewGlobalState()

	configPath := flag.String("config", "stalker.toml", "path to config file")

	flag.Parse()

	gs.Logger.Println("booting", *configPath)

	config, errs := ParseConfig(*configPath)
	for _, err := range errs {
		gs.Logger.Println(err)
	}

	if config == nil {
		gs.Logger.Fatalln("failed to start because configuration file is not valid")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	wg := &sync.WaitGroup{}
	for _, target := range config.Targets {
		wg.Add(1)
		go stalk(target, wg, gs)
	}

	<-sigChan

	gs.Logger.Println("shutting down")
	gs.cancel()

	wg.Wait()
}

func stalk(target *Target, wg *sync.WaitGroup, gs *GlobalState) {
	defer wg.Done()

	logPrefix := fmt.Sprint("stalk(", target.Name, "@", target.Period, " seconds)")

	gs.Logger.Println(logPrefix, "running")

	tick := time.After(time.Duration(target.Period) * time.Second)

	for {
		select {
		case <-gs.ctx.Done():
			gs.Logger.Println(logPrefix, "stopping")
			return
		case <-tick:
			req, err := http.NewRequestWithContext(gs.ctx, target.Method, target.Uri, nil)
			if err != nil {
				gs.Logger.Println(logPrefix, target.Method, target.Uri)
				gs.Logger.Println(logPrefix, "failed to construct req:", err)

				tick = time.After(time.Duration(target.Period) * time.Second)
				continue
			}

			resp, err := http.DefaultClient.Do(req)

			if err != nil {
				gs.Logger.Println(logPrefix, target.Method, target.Uri)
				gs.Logger.Println(logPrefix, "target is unhealthy:", err)
			} else if resp.StatusCode != target.ResponseCode {
				gs.Logger.Println(logPrefix, target.Method, target.Uri)
				gs.Logger.Println(logPrefix, "target is unhealthy: received status code", resp.StatusCode, "expected", target.ResponseCode)
			} else {
				gs.Logger.Println(logPrefix, "target is healthy")
			}

			tick = time.After(time.Duration(target.Period) * time.Second)
		}
	}
}