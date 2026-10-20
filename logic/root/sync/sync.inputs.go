package sync

import (
	modelsync "tty.fail/mrus/korda/models/sync"
)

type StreamInput struct {
	Types []modelsync.RequestType `json:"types"`
	Reset *bool                   `json:"reset"`
}

type AckSetInput struct {
	Acks []string `json:"acks" v:"max=1000"`
}

type AckDeleteInput struct {
	Types []modelsync.EntityType `json:"types" v:"optional"`
}
