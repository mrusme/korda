package repositories

import (
	"slices"

	"tty.fail/mrus/korda/services/repositories/apikey"
	"tty.fail/mrus/korda/services/repositories/session"
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
