package user

import (
	"context"

	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	return database.Row[user.User](ctx, repo.db, `INSERT INTO users (`+columns+`) VALUES (
		 $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		,$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) RETURNING `+columns,
		u.ID,
		u.Email,
		u.Password,
		u.PinCode,
		u.Name,
		u.IsAdmin,
		u.ShouldChangePassword,
		u.AvatarColor,
		u.ProfileImagePath,
		u.ProfileChangedAt,
		u.StorageLabel,
		u.QuotaSizeInBytes,
		u.QuotaUsageInBytes,
		u.Status,
		u.OAuthID,
		u.ClusterGroupID,
		u.CreatedAt,
		u.UpdatedAt,
		u.DeletedAt,
		u.UpdateID,
	)
}
