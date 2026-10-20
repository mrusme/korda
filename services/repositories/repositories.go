package repositories

import (
	"slices"

	"tty.fail/mrus/korda/services/repositories/apikey"
	"tty.fail/mrus/korda/services/repositories/asset"
	"tty.fail/mrus/korda/services/repositories/assetexif"
	"tty.fail/mrus/korda/services/repositories/assetfile"
	"tty.fail/mrus/korda/services/repositories/assetmetadata"
	"tty.fail/mrus/korda/services/repositories/session"
	reposync "tty.fail/mrus/korda/services/repositories/sync"
	"tty.fail/mrus/korda/services/repositories/synccheckpoint"
	"tty.fail/mrus/korda/services/repositories/systemmetadata"
	"tty.fail/mrus/korda/services/repositories/user"
	"tty.fail/mrus/korda/services/repositories/usermetadata"
	"tty.fail/mrus/korda/services/repositories/versionhistory"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Repositories struct {
	db             *database.Database
	User           *user.Repository
	UserMetadata   *usermetadata.Repository
	Session        *session.Repository
	APIKey         *apikey.Repository
	SystemMetadata *systemmetadata.Repository
	VersionHistory *versionhistory.Repository
	Asset          *asset.Repository
	AssetExif      *assetexif.Repository
	AssetFile      *assetfile.Repository
	AssetMetadata  *assetmetadata.Repository
	SyncCheckpoint *synccheckpoint.Repository
	Sync           *reposync.Repository
}

type startable interface {
	Startup() error
	Shutdown() error
}

func New(db *database.Database) (repos *Repositories, err error) {
	repos = new(Repositories)
	repos.db = db

	if repos.User, err = user.New(db); err != nil {
		return nil, err
	}
	if repos.UserMetadata, err = usermetadata.New(db); err != nil {
		return nil, err
	}
	if repos.Session, err = session.New(db); err != nil {
		return nil, err
	}
	if repos.APIKey, err = apikey.New(db); err != nil {
		return nil, err
	}
	if repos.SystemMetadata, err = systemmetadata.New(db); err != nil {
		return nil, err
	}
	if repos.VersionHistory, err = versionhistory.New(db); err != nil {
		return nil, err
	}
	if repos.Asset, err = asset.New(db); err != nil {
		return nil, err
	}
	if repos.AssetExif, err = assetexif.New(db); err != nil {
		return nil, err
	}
	if repos.AssetFile, err = assetfile.New(db); err != nil {
		return nil, err
	}
	if repos.AssetMetadata, err = assetmetadata.New(db); err != nil {
		return nil, err
	}
	if repos.SyncCheckpoint, err = synccheckpoint.New(db); err != nil {
		return nil, err
	}
	if repos.Sync, err = reposync.New(db); err != nil {
		return nil, err
	}

	return repos, nil
}

func (repos *Repositories) all() []startable {
	return []startable{
		repos.User,
		repos.UserMetadata,
		repos.Session,
		repos.APIKey,
		repos.SystemMetadata,
		repos.VersionHistory,
		repos.Asset,
		repos.AssetExif,
		repos.AssetFile,
		repos.AssetMetadata,
		repos.SyncCheckpoint,
		repos.Sync,
	}
}

func (repos *Repositories) Startup() error {
	for _, repo := range repos.all() {
		if err := repo.Startup(); err != nil {
			return err
		}
	}

	return nil
}

func (repos *Repositories) Shutdown() error {
	for _, repo := range slices.Backward(repos.all()) {
		if err := repo.Shutdown(); err != nil {
			return err
		}
	}

	return nil
}

func (repos *Repositories) DB() *database.Database {
	return repos.db
}
