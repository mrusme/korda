package routes

import (
	"strings"

	"xn--gckvb8fzb.com/glides/http/route"
)

const DefaultTitle string = "korda"

var Routes route.Table = route.Table{
	"Root":            {Hierarchy: []string{"root"}},
	"WellKnown":       {Hierarchy: []string{"root", ".well-known"}, NoBreadcrumb: true},
	"WellKnownImmich": {Hierarchy: []string{"root", ".well-known", "immich"}, NoBreadcrumb: true},
	"API":             {Hierarchy: []string{"root", "api"}, NoBreadcrumb: true},
	"APIActivities":   {Hierarchy: []string{"root", "api", "activities"}, NoBreadcrumb: true},
	"APIAdmin":        {Hierarchy: []string{"root", "api", "admin"}, NoBreadcrumb: true},
	"APIAdminUsers":   {Hierarchy: []string{"root", "api", "admin", "users"}, NoBreadcrumb: true},
	"APIAdminUser":    {Hierarchy: []string{"root", "api", "admin", "users", ":id"}, NoBreadcrumb: true},
	"APIAlbums":       {Hierarchy: []string{"root", "api", "albums"}, NoBreadcrumb: true},
	"APIAPIKeys":      {Hierarchy: []string{"root", "api", "api-keys"}, NoBreadcrumb: true},
	"APIAPIKey":       {Hierarchy: []string{"root", "api", "api-keys", ":id"}, NoBreadcrumb: true},
	"APIAssets":       {Hierarchy: []string{"root", "api", "assets"}, NoBreadcrumb: true},
	"APIAsset":        {Hierarchy: []string{"root", "api", "assets", ":id"}, NoBreadcrumb: true},
	"APIAuth":         {Hierarchy: []string{"root", "api", "auth"}, NoBreadcrumb: true},
	"APIAuthLogin":    {Hierarchy: []string{"root", "api", "auth", "login"}, NoBreadcrumb: true},
	"APIMemories":     {Hierarchy: []string{"root", "api", "memories"}, NoBreadcrumb: true},
	"APIPartners":     {Hierarchy: []string{"root", "api", "partners"}, NoBreadcrumb: true},
	"APISearch":       {Hierarchy: []string{"root", "api", "search"}, NoBreadcrumb: true},
	"APIServer":       {Hierarchy: []string{"root", "api", "server"}, NoBreadcrumb: true},
	"APIStacks":       {Hierarchy: []string{"root", "api", "stacks"}, NoBreadcrumb: true},
	"APISync":         {Hierarchy: []string{"root", "api", "sync"}, NoBreadcrumb: true},
	"APISessions":     {Hierarchy: []string{"root", "api", "sessions"}, NoBreadcrumb: true},
	"APISharedLinks":  {Hierarchy: []string{"root", "api", "shared-links"}, NoBreadcrumb: true},
	"APISession":      {Hierarchy: []string{"root", "api", "sessions", ":id"}, NoBreadcrumb: true},
	"APITags":         {Hierarchy: []string{"root", "api", "tags"}, NoBreadcrumb: true},
	"APITrash":        {Hierarchy: []string{"root", "api", "trash"}, NoBreadcrumb: true},
	"APIUsers":        {Hierarchy: []string{"root", "api", "users"}, NoBreadcrumb: true},
	"APIUsersMe":      {Hierarchy: []string{"root", "api", "users", "me"}, NoBreadcrumb: true},
	"APIUser":         {Hierarchy: []string{"root", "api", "users", ":id"}, NoBreadcrumb: true},
	"APIView":         {Hierarchy: []string{"root", "api", "view"}, NoBreadcrumb: true},
}

func APIPath(id string) string {
	url := strings.TrimPrefix(route.For(id).AsURL(), route.For("API").AsURL())
	return "/" + strings.TrimPrefix(url, "/")
}
