package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/apikey"
	"tty.fail/mrus/korda/models/permission"
)

type APIKey struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	CreatedAt   jsontime.Time           `json:"createdAt"`
	UpdatedAt   jsontime.Time           `json:"updatedAt"`
	Permissions []permission.Permission `json:"permissions"`
}

type APIKeyCreate struct {
	APIKey
	Secret string `json:"secret"`
	Key    APIKey `json:"apiKey"`
}

func NewAPIKey(k *apikey.APIKey) APIKey {
	permissions := []permission.Permission(k.Permissions)
	if permissions == nil {
		permissions = []permission.Permission{}
	}

	return APIKey{
		ID:          id(k.ID),
		Name:        k.Name,
		CreatedAt:   time3(k.CreatedAt),
		UpdatedAt:   time3(k.UpdatedAt),
		Permissions: permissions,
	}
}

func NewAPIKeys(list []apikey.APIKey) []APIKey {
	out := make([]APIKey, 0, len(list))
	for i := range list {
		out = append(out, NewAPIKey(&list[i]))
	}

	return out
}

func NewAPIKeyCreate(k *apikey.APIKey, secret string) APIKeyCreate {
	key := NewAPIKey(k)

	return APIKeyCreate{APIKey: key, Secret: secret, Key: key}
}
