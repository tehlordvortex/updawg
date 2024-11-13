package main

import (
	"context"
	"log"
)

type GlobalState struct {
	ctx    context.Context
	cancel context.CancelFunc
	Logger *log.Logger
}

func NewGlobalState() *GlobalState {
	ctx, cancel := context.WithCancel(context.Background())

	return &GlobalState{
		ctx:    ctx,
		cancel: cancel,
		Logger: log.Default(),
	}
}
