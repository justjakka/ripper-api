package ripper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hibiken/asynq"
)

const (
	TypeRip    = "download:apple"
	TypeInit   = "init:queue"
	TypeDelete = "remove:task"
)

type RipPayload struct {
	AlbumId    string
	Token      string
	Storefront string
	Wrapper    string
	WebDir     string
}

type DeletePayload struct {
	FolderPath string
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

func NewDeleteTask(dir string) (*asynq.Task, error) {
	payload, err := json.Marshal(DeletePayload{FolderPath: dir})

	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeDelete, payload), nil
}

func HandleProcessTask(_ context.Context, t *asynq.Task) error {
	var p RipPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	folder, err := Rip(p.AlbumId, p.Token, p.Storefront, p.Wrapper, p.WebDir)
	if err != nil {
		return err
	}

	res := []byte(folder)

	_, err = t.ResultWriter().Write(res)
	if err != nil {
		return err
	}
	return nil
}

func HandleInitQueueTask(_ context.Context, _ *asynq.Task) error {
	return nil
}

func HandleDeleteTask(_ context.Context, t *asynq.Task) error {
	var p DeletePayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	err := os.RemoveAll(p.FolderPath)
	if err != nil {
		return err
	}
	return nil
}
