package workflows

import (
	"encoding/json"
	"time"

	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/helpers/optional"
)

type Trigger string

func (Trigger) Enum() []string {
	return []string{"AssetCreate", "AssetMetadataExtraction", "AssetTagged"}
}

type Type string

func (Type) Enum() []string { return []string{"AssetV1"} }

type Result string

func (Result) Enum() []string { return []string{"completed", "halted", "error"} }

type PluginSearchInput struct {
	ID          *string `json:"id" v:"uuid"`
	Enabled     *bool   `json:"enabled" v:"boolean"`
	Name        *string `json:"name"`
	Version     *string `json:"version"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type MethodSearchInput struct {
	ID            *string  `json:"id" v:"uuid"`
	Enabled       *bool    `json:"enabled" v:"boolean"`
	Name          *string  `json:"name"`
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	Type          *Type    `json:"type"`
	Trigger       *Trigger `json:"trigger"`
	PluginName    *string  `json:"pluginName"`
	PluginVersion *string  `json:"pluginVersion"`
}

type SearchInput struct {
	ID          *string  `json:"id" v:"uuid"`
	Trigger     *Trigger `json:"trigger"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Enabled     *bool    `json:"enabled" v:"boolean"`
	Logging     *bool    `json:"logging" v:"boolean"`
}

type Step struct {
	Method  string                          `json:"method"`
	Config  optional.Value[json.RawMessage] `json:"config" v:"required,nullable"`
	Enabled *bool                           `json:"enabled"`
}

type CreateInput struct {
	Trigger     Trigger                `json:"trigger"`
	Name        optional.Value[string] `json:"name" v:"nullable"`
	Description optional.Value[string] `json:"description" v:"nullable"`
	Enabled     *bool                  `json:"enabled"`
	Logging     *bool                  `json:"logging"`
	Steps       []Step                 `json:"steps" v:"optional"`
}

type UpdateInput struct {
	Trigger     *Trigger               `json:"trigger"`
	Name        optional.Value[string] `json:"name" v:"nullable"`
	Description optional.Value[string] `json:"description" v:"nullable"`
	Enabled     *bool                  `json:"enabled"`
	Logging     *bool                  `json:"logging"`
	Steps       []Step                 `json:"steps" v:"optional"`
}

type LogsInput struct {
	Result *Result    `json:"result"`
	Before *time.Time `json:"before" v:"datetime"`
	Limit  *int64     `json:"limit" v:"coerce,positive"`
}

func steps(list []Step) error {
	for _, s := range list {
		return &errs.UnknownMethod{Method: s.Method}
	}

	return errs.UnsupportedWorkflows
}

func Create(in *CreateInput) error {
	return steps(in.Steps)
}

func MethodSearch(in *MethodSearchInput) error {
	if in.Type != nil {
		return errs.ErrPluginTypeFilter
	}

	return nil
}
