package ripper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

const (
	TypeRip = "download:apple"
)

type RipPayload struct {
	AlbumId     string
	Token       string
	Storefront  string
	WrapperPort uint
	WebDir      string
}

func NewRipTask(storefront string, albumId string, port uint, webdir string) (*asynq.Task, error) {
	token, err := getToken()
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(RipPayload{AlbumId: albumId, Token: token, Storefront: storefront, WrapperPort: port, WebDir: webdir})

	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeRip, payload), nil
}

func HandleRipTask(ctx context.Context, t *asynq.Task) error {
	var p RipPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Printf("Started processing album %v", p.AlbumId)

	if err := Rip(p.AlbumId, p.Token, p.Storefront, p.WrapperPort, p.WebDir); err != nil {
		return err
	}

	return nil
}
