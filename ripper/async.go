package ripper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeRip  = "download:apple"
	TypeInit = "init:queue"
)

type RipPayload struct {
	AlbumId    string
	Token      string
	Storefront string
	Wrapper    string
	WebDir     string
}

func NewRipTask(storefront string, albumId string, webdir string, wrapper string) (*asynq.Task, error) {
	token, err := getToken()
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(RipPayload{AlbumId: albumId, Token: token, Storefront: storefront, Wrapper: wrapper, WebDir: webdir})

	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeRip, payload), nil
}

func NewInitQueueTask() (*asynq.Task, error) {
	var payload []byte

	return asynq.NewTask(TypeInit, payload), nil
}

func HandleProcessTask(ctx context.Context, t *asynq.Task) error {
	var p RipPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if err := Rip(p.AlbumId, p.Token, p.Storefront, p.Wrapper, p.WebDir); err != nil {
		return err
	}

	return nil
}

func HandleInitQueueTask(ctx context.Context, t *asynq.Task) error {
	return nil
}
