package view

import (
	"tty.fail/mrus/korda/helpers/jsontime"
	"tty.fail/mrus/korda/models/user"
)

type User struct {
	ID               string           `json:"id"`
	Email            string           `json:"email"`
	Name             string           `json:"name"`
	ProfileImagePath string           `json:"profileImagePath"`
	AvatarColor      user.AvatarColor `json:"avatarColor"`
	ProfileChangedAt jsontime.Time    `json:"profileChangedAt"`
}

type UserLicense struct {
	LicenseKey    string        `json:"licenseKey"`
	ActivationKey string        `json:"activationKey"`
	ActivatedAt   jsontime.Time `json:"activatedAt"`
}

type UserAdmin struct {
	ID                   string           `json:"id"`
	Email                string           `json:"email"`
	Name                 string           `json:"name"`
	ProfileImagePath     string           `json:"profileImagePath"`
	AvatarColor          user.AvatarColor `json:"avatarColor"`
	ProfileChangedAt     jsontime.Time    `json:"profileChangedAt"`
	ClusterGroupID       string           `json:"clusterGroupId"`
	StorageLabel         *string          `json:"storageLabel"`
	ShouldChangePassword bool             `json:"shouldChangePassword"`
	IsAdmin              bool             `json:"isAdmin"`
	CreatedAt            jsontime.Time    `json:"createdAt"`
	DeletedAt            *jsontime.Time   `json:"deletedAt"`
	UpdatedAt            jsontime.Time    `json:"updatedAt"`
	OAuthID              string           `json:"oauthId"`
	QuotaSizeInBytes     *int64           `json:"quotaSizeInBytes"`
	QuotaUsageInBytes    int64            `json:"quotaUsageInBytes"`
	Status               user.Status      `json:"status"`
	License              *UserLicense     `json:"license"`
}

type Onboarding struct {
	IsOnboarded bool `json:"isOnboarded"`
}

type ProfileImage struct {
	UserID           string        `json:"userId"`
	ProfileImagePath string        `json:"profileImagePath"`
	ProfileChangedAt jsontime.Time `json:"profileChangedAt"`
}

func NewUser(u *user.User) User {
	return User{
		ID:               id(u.ID),
		Email:            u.Email,
		Name:             u.Name,
		ProfileImagePath: u.ProfileImagePath,
		AvatarColor:      u.Color(),
		ProfileChangedAt: time3(u.ProfileChangedAt),
	}
}

func NewUsers(list []user.User) []User {
	out := make([]User, 0, len(list))
	for i := range list {
		out = append(out, NewUser(&list[i]))
	}

	return out
}

func NewUserAdmin(u *user.User) UserAdmin {
	return UserAdmin{
		ID:                   id(u.ID),
		Email:                u.Email,
		Name:                 u.Name,
		ProfileImagePath:     u.ProfileImagePath,
		AvatarColor:          u.Color(),
		ProfileChangedAt:     time3(u.ProfileChangedAt),
		ClusterGroupID:       id(u.ClusterGroupID),
		StorageLabel:         nullable(u.StorageLabel),
		ShouldChangePassword: u.ShouldChangePassword,
		IsAdmin:              u.IsAdmin,
		CreatedAt:            time3(u.CreatedAt),
		DeletedAt:            jsontime.Nullable(u.DeletedAt),
		UpdatedAt:            time3(u.UpdatedAt),
		OAuthID:              u.OAuthID.V,
		QuotaSizeInBytes:     nullable(u.QuotaSizeInBytes),
		QuotaUsageInBytes:    u.QuotaUsageInBytes,
		Status:               u.Status,
		License:              nil,
	}
}

func NewUserAdmins(list []user.User) []UserAdmin {
	out := make([]UserAdmin, 0, len(list))
	for i := range list {
		out = append(out, NewUserAdmin(&list[i]))
	}

	return out
}

func NewProfileImage(u *user.User) ProfileImage {
	return ProfileImage{
		UserID:           id(u.ID),
		ProfileImagePath: u.ProfileImagePath,
		ProfileChangedAt: time3(u.ProfileChangedAt),
	}
}
