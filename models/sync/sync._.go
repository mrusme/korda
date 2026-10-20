package sync

import "strings"

type RequestType string

const (
	ReqAlbumsV1            RequestType = "AlbumsV1"
	ReqAlbumsV2            RequestType = "AlbumsV2"
	ReqAlbumUsersV1        RequestType = "AlbumUsersV1"
	ReqAlbumToAssetsV1     RequestType = "AlbumToAssetsV1"
	ReqAlbumAssetsV1       RequestType = "AlbumAssetsV1"
	ReqAlbumAssetsV2       RequestType = "AlbumAssetsV2"
	ReqAlbumAssetExifsV1   RequestType = "AlbumAssetExifsV1"
	ReqAssetsV1            RequestType = "AssetsV1"
	ReqAssetsV2            RequestType = "AssetsV2"
	ReqAssetExifsV1        RequestType = "AssetExifsV1"
	ReqAssetEditsV1        RequestType = "AssetEditsV1"
	ReqAssetMetadataV1     RequestType = "AssetMetadataV1"
	ReqAssetOcrV1          RequestType = "AssetOcrV1"
	ReqAuthUsersV1         RequestType = "AuthUsersV1"
	ReqAuthUsersV2         RequestType = "AuthUsersV2"
	ReqMemoriesV1          RequestType = "MemoriesV1"
	ReqMemoriesV2          RequestType = "MemoriesV2"
	ReqMemoryToAssetsV1    RequestType = "MemoryToAssetsV1"
	ReqMemoryToAssetsV2    RequestType = "MemoryToAssetsV2"
	ReqPartnersV1          RequestType = "PartnersV1"
	ReqPartnerAssetsV1     RequestType = "PartnerAssetsV1"
	ReqPartnerAssetsV2     RequestType = "PartnerAssetsV2"
	ReqPartnerAssetExifsV1 RequestType = "PartnerAssetExifsV1"
	ReqPartnerStacksV1     RequestType = "PartnerStacksV1"
	ReqStacksV1            RequestType = "StacksV1"
	ReqUsersV1             RequestType = "UsersV1"
	ReqPeopleV1            RequestType = "PeopleV1"
	ReqAssetFacesV1        RequestType = "AssetFacesV1"
	ReqAssetFacesV2        RequestType = "AssetFacesV2"
	ReqAssetFacesV3        RequestType = "AssetFacesV3"
	ReqUserMetadataV1      RequestType = "UserMetadataV1"
)

var RequestTypes = []RequestType{
	ReqAlbumsV1, ReqAlbumsV2, ReqAlbumUsersV1, ReqAlbumToAssetsV1, ReqAlbumAssetsV1, ReqAlbumAssetsV2, ReqAlbumAssetExifsV1,
	ReqAssetsV1, ReqAssetsV2, ReqAssetExifsV1, ReqAssetEditsV1, ReqAssetMetadataV1, ReqAssetOcrV1, ReqAuthUsersV1, ReqAuthUsersV2,
	ReqMemoriesV1, ReqMemoriesV2, ReqMemoryToAssetsV1, ReqMemoryToAssetsV2, ReqPartnersV1, ReqPartnerAssetsV1, ReqPartnerAssetsV2,
	ReqPartnerAssetExifsV1, ReqPartnerStacksV1, ReqStacksV1, ReqUsersV1, ReqPeopleV1, ReqAssetFacesV1, ReqAssetFacesV2, ReqAssetFacesV3,
	ReqUserMetadataV1,
}

func (RequestType) Enum() []string {
	out := make([]string, len(RequestTypes))
	for i, t := range RequestTypes {
		out[i] = string(t)
	}

	return out
}

var Order = []RequestType{
	ReqAuthUsersV1, ReqAuthUsersV2, ReqUsersV1, ReqPartnersV1, ReqAssetsV1, ReqAssetsV2, ReqStacksV1, ReqPartnerAssetsV1, ReqPartnerAssetsV2,
	ReqPartnerStacksV1, ReqAlbumAssetsV1, ReqAlbumAssetsV2, ReqAlbumsV1, ReqAlbumsV2, ReqAlbumUsersV1, ReqAlbumToAssetsV1, ReqAssetExifsV1,
	ReqAlbumAssetExifsV1, ReqAssetOcrV1, ReqPartnerAssetExifsV1, ReqMemoriesV1, ReqMemoriesV2, ReqMemoryToAssetsV1, ReqMemoryToAssetsV2,
	ReqPeopleV1, ReqAssetFacesV1, ReqAssetFacesV2, ReqAssetFacesV3, ReqUserMetadataV1, ReqAssetMetadataV1, ReqAssetEditsV1,
}

var Deprecated = map[RequestType]RequestType{
	ReqAssetsV1:        ReqAssetsV2,
	ReqAssetFacesV1:    ReqAssetFacesV2,
	ReqPartnerAssetsV1: ReqPartnerAssetsV2,
	ReqAlbumAssetsV1:   ReqAlbumAssetsV2,
}

type EntityType string

const (
	AuthUserV1                 EntityType = "AuthUserV1"
	AuthUserV2                 EntityType = "AuthUserV2"
	UserV1                     EntityType = "UserV1"
	UserDeleteV1               EntityType = "UserDeleteV1"
	AssetV1                    EntityType = "AssetV1"
	AssetV2                    EntityType = "AssetV2"
	AssetDeleteV1              EntityType = "AssetDeleteV1"
	AssetExifV1                EntityType = "AssetExifV1"
	AssetEditV1                EntityType = "AssetEditV1"
	AssetEditDeleteV1          EntityType = "AssetEditDeleteV1"
	AssetMetadataV1            EntityType = "AssetMetadataV1"
	AssetMetadataDeleteV1      EntityType = "AssetMetadataDeleteV1"
	AssetOcrV1                 EntityType = "AssetOcrV1"
	AssetOcrDeleteV1           EntityType = "AssetOcrDeleteV1"
	PartnerV1                  EntityType = "PartnerV1"
	PartnerDeleteV1            EntityType = "PartnerDeleteV1"
	PartnerAssetV1             EntityType = "PartnerAssetV1"
	PartnerAssetV2             EntityType = "PartnerAssetV2"
	PartnerAssetBackfillV1     EntityType = "PartnerAssetBackfillV1"
	PartnerAssetBackfillV2     EntityType = "PartnerAssetBackfillV2"
	PartnerAssetDeleteV1       EntityType = "PartnerAssetDeleteV1"
	PartnerAssetExifV1         EntityType = "PartnerAssetExifV1"
	PartnerAssetExifBackfillV1 EntityType = "PartnerAssetExifBackfillV1"
	PartnerStackBackfillV1     EntityType = "PartnerStackBackfillV1"
	PartnerStackDeleteV1       EntityType = "PartnerStackDeleteV1"
	PartnerStackV1             EntityType = "PartnerStackV1"
	AlbumV1                    EntityType = "AlbumV1"
	AlbumV2                    EntityType = "AlbumV2"
	AlbumDeleteV1              EntityType = "AlbumDeleteV1"
	AlbumUserV1                EntityType = "AlbumUserV1"
	AlbumUserBackfillV1        EntityType = "AlbumUserBackfillV1"
	AlbumUserDeleteV1          EntityType = "AlbumUserDeleteV1"
	AlbumAssetCreateV1         EntityType = "AlbumAssetCreateV1"
	AlbumAssetCreateV2         EntityType = "AlbumAssetCreateV2"
	AlbumAssetUpdateV1         EntityType = "AlbumAssetUpdateV1"
	AlbumAssetUpdateV2         EntityType = "AlbumAssetUpdateV2"
	AlbumAssetBackfillV1       EntityType = "AlbumAssetBackfillV1"
	AlbumAssetBackfillV2       EntityType = "AlbumAssetBackfillV2"
	AlbumAssetExifCreateV1     EntityType = "AlbumAssetExifCreateV1"
	AlbumAssetExifUpdateV1     EntityType = "AlbumAssetExifUpdateV1"
	AlbumAssetExifBackfillV1   EntityType = "AlbumAssetExifBackfillV1"
	AlbumToAssetV1             EntityType = "AlbumToAssetV1"
	AlbumToAssetDeleteV1       EntityType = "AlbumToAssetDeleteV1"
	AlbumToAssetBackfillV1     EntityType = "AlbumToAssetBackfillV1"
	MemoryV1                   EntityType = "MemoryV1"
	MemoryV2                   EntityType = "MemoryV2"
	MemoryDeleteV1             EntityType = "MemoryDeleteV1"
	MemoryToAssetV1            EntityType = "MemoryToAssetV1"
	MemoryToAssetV2            EntityType = "MemoryToAssetV2"
	MemoryToAssetDeleteV1      EntityType = "MemoryToAssetDeleteV1"
	StackV1                    EntityType = "StackV1"
	StackDeleteV1              EntityType = "StackDeleteV1"
	PersonV1                   EntityType = "PersonV1"
	PersonDeleteV1             EntityType = "PersonDeleteV1"
	AssetFaceV1                EntityType = "AssetFaceV1"
	AssetFaceV2                EntityType = "AssetFaceV2"
	AssetFaceV3                EntityType = "AssetFaceV3"
	AssetFaceDeleteV1          EntityType = "AssetFaceDeleteV1"
	UserMetadataV1             EntityType = "UserMetadataV1"
	UserMetadataDeleteV1       EntityType = "UserMetadataDeleteV1"
	SyncAckV1                  EntityType = "SyncAckV1"
	SyncResetV1                EntityType = "SyncResetV1"
	SyncCompleteV1             EntityType = "SyncCompleteV1"
)

var EntityTypes = []EntityType{
	AuthUserV1, AuthUserV2, UserV1, UserDeleteV1, AssetV1, AssetV2, AssetDeleteV1, AssetExifV1, AssetEditV1,
	AssetEditDeleteV1, AssetMetadataV1, AssetMetadataDeleteV1, AssetOcrV1, AssetOcrDeleteV1, PartnerV1,
	PartnerDeleteV1, PartnerAssetV1, PartnerAssetV2, PartnerAssetBackfillV1, PartnerAssetBackfillV2,
	PartnerAssetDeleteV1, PartnerAssetExifV1, PartnerAssetExifBackfillV1, PartnerStackBackfillV1,
	PartnerStackDeleteV1, PartnerStackV1, AlbumV1, AlbumV2, AlbumDeleteV1, AlbumUserV1, AlbumUserBackfillV1,
	AlbumUserDeleteV1, AlbumAssetCreateV1, AlbumAssetCreateV2, AlbumAssetUpdateV1, AlbumAssetUpdateV2,
	AlbumAssetBackfillV1, AlbumAssetBackfillV2, AlbumAssetExifCreateV1, AlbumAssetExifUpdateV1,
	AlbumAssetExifBackfillV1, AlbumToAssetV1, AlbumToAssetDeleteV1, AlbumToAssetBackfillV1, MemoryV1, MemoryV2,
	MemoryDeleteV1, MemoryToAssetV1, MemoryToAssetV2, MemoryToAssetDeleteV1, StackV1, StackDeleteV1, PersonV1,
	PersonDeleteV1, AssetFaceV1, AssetFaceV2, AssetFaceV3, AssetFaceDeleteV1, UserMetadataV1, UserMetadataDeleteV1,
	SyncAckV1, SyncResetV1, SyncCompleteV1,
}

func (EntityType) Enum() []string {
	out := make([]string, len(EntityTypes))
	for i, t := range EntityTypes {
		out[i] = string(t)
	}

	return out
}

func IsEntityType(s string) bool {
	for _, t := range EntityTypes {
		if string(t) == s {
			return true
		}
	}

	return false
}

const (
	ResetID    = "reset"
	CompleteID = "complete"
)

type Ack struct {
	Type     string
	UpdateID *string
	ExtraID  *string
}

func ParseAck(ack string) Ack {
	parts := strings.Split(ack, "|")
	a := Ack{Type: parts[0]}
	if len(parts) > 1 {
		a.UpdateID = &parts[1]
	}
	if len(parts) > 2 {
		a.ExtraID = &parts[2]
	}

	return a
}

func FormatAck(t EntityType, ids ...string) string {
	return strings.Join(append([]string{string(t)}, ids...), "|")
}
