package users

import (
	"context"

	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/helpers/optional"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	Email                string                           `json:"email" v:"email"`
	Password             string                           `json:"password"`
	Name                 string                           `json:"name"`
	AvatarColor          optional.Value[user.AvatarColor] `json:"avatarColor" v:"nullable"`
	PinCode              optional.Value[string]           `json:"pinCode" v:"nullable,regex=pin"`
	StorageLabel         optional.Value[string]           `json:"storageLabel" v:"nullable,sanitize"`
	QuotaSizeInBytes     optional.Value[int64]            `json:"quotaSizeInBytes" v:"nullable,min=0"`
	ShouldChangePassword *bool                            `json:"shouldChangePassword"`
	Notify               *bool                            `json:"notify"`
	IsAdmin              *bool                            `json:"isAdmin"`
}

func Create(ctx context.Context, rt *runtime.Runtime, in *CreateInput) (*user.User, error) {
	if in.Password == "" {
		return nil, errs.ErrPasswordRequired
	}

	create := logicusers.CreateInput{
		Email:            in.Email,
		Password:         in.Password,
		Name:             in.Name,
		IsAdmin:          in.IsAdmin != nil && *in.IsAdmin,
		AvatarColor:      in.AvatarColor,
		PinCode:          in.PinCode,
		StorageLabel:     in.StorageLabel,
		QuotaSizeInBytes: in.QuotaSizeInBytes,
	}
	if in.ShouldChangePassword != nil {
		create.ShouldChangePassword = optional.Of(*in.ShouldChangePassword)
	}

	return logicusers.Create(ctx, rt, create)
}
