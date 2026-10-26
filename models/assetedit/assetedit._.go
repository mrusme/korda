package assetedit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Action string

const (
	ActionCrop   Action = "crop"
	ActionRotate Action = "rotate"
	ActionMirror Action = "mirror"
)

func (Action) Enum() []string {
	return []string{string(ActionCrop), string(ActionRotate), string(ActionMirror)}
}

type Axis string

const (
	AxisHorizontal Axis = "horizontal"
	AxisVertical   Axis = "vertical"
)

func (Axis) Enum() []string {
	return []string{string(AxisHorizontal), string(AxisVertical)}
}

type Crop struct {
	X      int64 `json:"x" v:"min=0"`
	Y      int64 `json:"y" v:"min=0"`
	Width  int64 `json:"width" v:"min=1"`
	Height int64 `json:"height" v:"min=1"`
}

type Rotate struct {
	Angle int64 `json:"angle" v:"min=0,max=270,refine=angle"`
}

type Mirror struct {
	Axis Axis `json:"axis"`
}

type Input struct {
	Action     Action          `json:"action"`
	Parameters json.RawMessage `json:"parameters"`
}

type Inputs []Input

type Edit struct {
	ID         uuid.UUID
	AssetID    uuid.UUID
	Action     Action
	Parameters string
	Sequence   int64
	UpdatedAt  time.Time
	UpdateID   uuid.UUID
}
