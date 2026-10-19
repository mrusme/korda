package permission

type Permission string

const (
	All                       Permission = "all"
	ActivityCreate            Permission = "activity.create"
	ActivityRead              Permission = "activity.read"
	ActivityUpdate            Permission = "activity.update"
	ActivityDelete            Permission = "activity.delete"
	ActivityStatistics        Permission = "activity.statistics"
	APIKeyCreate              Permission = "apiKey.create"
	APIKeyRead                Permission = "apiKey.read"
	APIKeyUpdate              Permission = "apiKey.update"
	APIKeyDelete              Permission = "apiKey.delete"
	APIKeyRotate              Permission = "apiKey.rotate"
	AssetRead                 Permission = "asset.read"
	AssetUpdate               Permission = "asset.update"
	AssetDelete               Permission = "asset.delete"
	AssetStatistics           Permission = "asset.statistics"
	AssetShare                Permission = "asset.share"
	AssetView                 Permission = "asset.view"
	AssetDownload             Permission = "asset.download"
	AssetUpload               Permission = "asset.upload"
	AssetCopy                 Permission = "asset.copy"
	AssetDerive               Permission = "asset.derive"
	AssetFileRead             Permission = "assetFile.read"
	AssetFileDelete           Permission = "assetFile.delete"
	AssetFileDownload         Permission = "assetFile.download"
	AssetEditGet              Permission = "asset.edit.get"
	AssetEditCreate           Permission = "asset.edit.create"
	AssetEditDelete           Permission = "asset.edit.delete"
	AlbumCreate               Permission = "album.create"
	AlbumRead                 Permission = "album.read"
	AlbumUpdate               Permission = "album.update"
	AlbumDelete               Permission = "album.delete"
	AlbumStatistics           Permission = "album.statistics"
	AlbumShare                Permission = "album.share"
	AlbumDownload             Permission = "album.download"
	AlbumAssetCreate          Permission = "albumAsset.create"
	AlbumAssetDelete          Permission = "albumAsset.delete"
	AlbumUserCreate           Permission = "albumUser.create"
	AlbumUserUpdate           Permission = "albumUser.update"
	AlbumUserDelete           Permission = "albumUser.delete"
	AuthChangePassword        Permission = "auth.changePassword"
	AuthDeviceDelete          Permission = "authDevice.delete"
	ArchiveRead               Permission = "archive.read"
	BackupList                Permission = "backup.list"
	BackupDownload            Permission = "backup.download"
	BackupUpload              Permission = "backup.upload"
	BackupDelete              Permission = "backup.delete"
	ClusterGroupRead          Permission = "clusterGroup.read"
	ClusterGroupLeave         Permission = "clusterGroup.leave"
	ClusterGroupRequestCreate Permission = "clusterGroupRequest.create"
	ClusterGroupRequestRead   Permission = "clusterGroupRequest.read"
	ClusterGroupRequestDelete Permission = "clusterGroupRequest.delete"
	AdminConfigRead           Permission = "adminConfig.read"
	AdminConfigUpdate         Permission = "adminConfig.update"
	UserConfigRead            Permission = "userConfig.read"
	DuplicateRead             Permission = "duplicate.read"
	DuplicateDelete           Permission = "duplicate.delete"
	FaceCreate                Permission = "face.create"
	FaceRead                  Permission = "face.read"
	FaceUpdate                Permission = "face.update"
	FaceDelete                Permission = "face.delete"
	FolderRead                Permission = "folder.read"
	JobCreate                 Permission = "job.create"
	JobRead                   Permission = "job.read"
	LibraryCreate             Permission = "library.create"
	LibraryRead               Permission = "library.read"
	LibraryUpdate             Permission = "library.update"
	LibraryDelete             Permission = "library.delete"
	LibraryStatistics         Permission = "library.statistics"
	TimelineRead              Permission = "timeline.read"
	TimelineDownload          Permission = "timeline.download"
	Maintenance               Permission = "maintenance"
	MapRead                   Permission = "map.read"
	MapSearch                 Permission = "map.search"
	MemoryCreate              Permission = "memory.create"
	MemoryRead                Permission = "memory.read"
	MemoryUpdate              Permission = "memory.update"
	MemoryDelete              Permission = "memory.delete"
	MemoryStatistics          Permission = "memory.statistics"
	MemoryAssetCreate         Permission = "memoryAsset.create"
	MemoryAssetDelete         Permission = "memoryAsset.delete"
	NotificationCreate        Permission = "notification.create"
	NotificationRead          Permission = "notification.read"
	NotificationUpdate        Permission = "notification.update"
	NotificationDelete        Permission = "notification.delete"
	PartnerCreate             Permission = "partner.create"
	PartnerRead               Permission = "partner.read"
	PartnerUpdate             Permission = "partner.update"
	PartnerDelete             Permission = "partner.delete"
	PersonCreate              Permission = "person.create"
	PersonRead                Permission = "person.read"
	PersonUpdate              Permission = "person.update"
	PersonDelete              Permission = "person.delete"
	PersonStatistics          Permission = "person.statistics"
	PersonMerge               Permission = "person.merge"
	PersonReassign            Permission = "person.reassign"
	PinCodeCreate             Permission = "pinCode.create"
	PinCodeUpdate             Permission = "pinCode.update"
	PinCodeDelete             Permission = "pinCode.delete"
	PluginCreate              Permission = "plugin.create"
	PluginRead                Permission = "plugin.read"
	PluginUpdate              Permission = "plugin.update"
	PluginDelete              Permission = "plugin.delete"
	ServerAbout               Permission = "server.about"
	ServerApkLinks            Permission = "server.apkLinks"
	ServerStorage             Permission = "server.storage"
	ServerStatistics          Permission = "server.statistics"
	ServerVersionCheck        Permission = "server.versionCheck"
	ServerLicenseRead         Permission = "serverLicense.read"
	ServerLicenseUpdate       Permission = "serverLicense.update"
	ServerLicenseDelete       Permission = "serverLicense.delete"
	SessionCreate             Permission = "session.create"
	SessionRead               Permission = "session.read"
	SessionUpdate             Permission = "session.update"
	SessionDelete             Permission = "session.delete"
	SessionLock               Permission = "session.lock"
	SharedLinkCreate          Permission = "sharedLink.create"
	SharedLinkRead            Permission = "sharedLink.read"
	SharedLinkUpdate          Permission = "sharedLink.update"
	SharedLinkDelete          Permission = "sharedLink.delete"
	StackCreate               Permission = "stack.create"
	StackRead                 Permission = "stack.read"
	StackUpdate               Permission = "stack.update"
	StackDelete               Permission = "stack.delete"
	SyncStream                Permission = "sync.stream"
	SyncCheckpointRead        Permission = "syncCheckpoint.read"
	SyncCheckpointUpdate      Permission = "syncCheckpoint.update"
	SyncCheckpointDelete      Permission = "syncCheckpoint.delete"
	SystemConfigRead          Permission = "systemConfig.read"
	SystemConfigUpdate        Permission = "systemConfig.update"
	SystemMetadataRead        Permission = "systemMetadata.read"
	SystemMetadataUpdate      Permission = "systemMetadata.update"
	TagCreate                 Permission = "tag.create"
	TagRead                   Permission = "tag.read"
	TagUpdate                 Permission = "tag.update"
	TagDelete                 Permission = "tag.delete"
	TagAsset                  Permission = "tag.asset"
	UserRead                  Permission = "user.read"
	UserUpdate                Permission = "user.update"
	UserLicenseCreate         Permission = "userLicense.create"
	UserLicenseRead           Permission = "userLicense.read"
	UserLicenseUpdate         Permission = "userLicense.update"
	UserLicenseDelete         Permission = "userLicense.delete"
	UserOnboardingRead        Permission = "userOnboarding.read"
	UserOnboardingUpdate      Permission = "userOnboarding.update"
	UserOnboardingDelete      Permission = "userOnboarding.delete"
	UserPreferenceRead        Permission = "userPreference.read"
	UserPreferenceUpdate      Permission = "userPreference.update"
	UserProfileImageCreate    Permission = "userProfileImage.create"
	UserProfileImageRead      Permission = "userProfileImage.read"
	UserProfileImageUpdate    Permission = "userProfileImage.update"
	UserProfileImageDelete    Permission = "userProfileImage.delete"
	QueueRead                 Permission = "queue.read"
	QueueUpdate               Permission = "queue.update"
	QueueJobCreate            Permission = "queueJob.create"
	QueueJobRead              Permission = "queueJob.read"
	QueueJobUpdate            Permission = "queueJob.update"
	QueueJobDelete            Permission = "queueJob.delete"
	WorkflowCreate            Permission = "workflow.create"
	WorkflowRead              Permission = "workflow.read"
	WorkflowUpdate            Permission = "workflow.update"
	WorkflowDelete            Permission = "workflow.delete"
	WorkflowLogs              Permission = "workflow.logs"
	AdminUserCreate           Permission = "adminUser.create"
	AdminUserRead             Permission = "adminUser.read"
	AdminUserUpdate           Permission = "adminUser.update"
	AdminUserDelete           Permission = "adminUser.delete"
	AdminSessionRead          Permission = "adminSession.read"
	AdminAuthUnlinkAll        Permission = "adminAuth.unlinkAll"
)

var values = []string{
	string(All),
	string(ActivityCreate),
	string(ActivityRead),
	string(ActivityUpdate),
	string(ActivityDelete),
	string(ActivityStatistics),
	string(APIKeyCreate),
	string(APIKeyRead),
	string(APIKeyUpdate),
	string(APIKeyDelete),
	string(APIKeyRotate),
	string(AssetRead),
	string(AssetUpdate),
	string(AssetDelete),
	string(AssetStatistics),
	string(AssetShare),
	string(AssetView),
	string(AssetDownload),
	string(AssetUpload),
	string(AssetCopy),
	string(AssetDerive),
	string(AssetFileRead),
	string(AssetFileDelete),
	string(AssetFileDownload),
	string(AssetEditGet),
	string(AssetEditCreate),
	string(AssetEditDelete),
	string(AlbumCreate),
	string(AlbumRead),
	string(AlbumUpdate),
	string(AlbumDelete),
	string(AlbumStatistics),
	string(AlbumShare),
	string(AlbumDownload),
	string(AlbumAssetCreate),
	string(AlbumAssetDelete),
	string(AlbumUserCreate),
	string(AlbumUserUpdate),
	string(AlbumUserDelete),
	string(AuthChangePassword),
	string(AuthDeviceDelete),
	string(ArchiveRead),
	string(BackupList),
	string(BackupDownload),
	string(BackupUpload),
	string(BackupDelete),
	string(ClusterGroupRead),
	string(ClusterGroupLeave),
	string(ClusterGroupRequestCreate),
	string(ClusterGroupRequestRead),
	string(ClusterGroupRequestDelete),
	string(AdminConfigRead),
	string(AdminConfigUpdate),
	string(UserConfigRead),
	string(DuplicateRead),
	string(DuplicateDelete),
	string(FaceCreate),
	string(FaceRead),
	string(FaceUpdate),
	string(FaceDelete),
	string(FolderRead),
	string(JobCreate),
	string(JobRead),
	string(LibraryCreate),
	string(LibraryRead),
	string(LibraryUpdate),
	string(LibraryDelete),
	string(LibraryStatistics),
	string(TimelineRead),
	string(TimelineDownload),
	string(Maintenance),
	string(MapRead),
	string(MapSearch),
	string(MemoryCreate),
	string(MemoryRead),
	string(MemoryUpdate),
	string(MemoryDelete),
	string(MemoryStatistics),
	string(MemoryAssetCreate),
	string(MemoryAssetDelete),
	string(NotificationCreate),
	string(NotificationRead),
	string(NotificationUpdate),
	string(NotificationDelete),
	string(PartnerCreate),
	string(PartnerRead),
	string(PartnerUpdate),
	string(PartnerDelete),
	string(PersonCreate),
	string(PersonRead),
	string(PersonUpdate),
	string(PersonDelete),
	string(PersonStatistics),
	string(PersonMerge),
	string(PersonReassign),
	string(PinCodeCreate),
	string(PinCodeUpdate),
	string(PinCodeDelete),
	string(PluginCreate),
	string(PluginRead),
	string(PluginUpdate),
	string(PluginDelete),
	string(ServerAbout),
	string(ServerApkLinks),
	string(ServerStorage),
	string(ServerStatistics),
	string(ServerVersionCheck),
	string(ServerLicenseRead),
	string(ServerLicenseUpdate),
	string(ServerLicenseDelete),
	string(SessionCreate),
	string(SessionRead),
	string(SessionUpdate),
	string(SessionDelete),
	string(SessionLock),
	string(SharedLinkCreate),
	string(SharedLinkRead),
	string(SharedLinkUpdate),
	string(SharedLinkDelete),
	string(StackCreate),
	string(StackRead),
	string(StackUpdate),
	string(StackDelete),
	string(SyncStream),
	string(SyncCheckpointRead),
	string(SyncCheckpointUpdate),
	string(SyncCheckpointDelete),
	string(SystemConfigRead),
	string(SystemConfigUpdate),
	string(SystemMetadataRead),
	string(SystemMetadataUpdate),
	string(TagCreate),
	string(TagRead),
	string(TagUpdate),
	string(TagDelete),
	string(TagAsset),
	string(UserRead),
	string(UserUpdate),
	string(UserLicenseCreate),
	string(UserLicenseRead),
	string(UserLicenseUpdate),
	string(UserLicenseDelete),
	string(UserOnboardingRead),
	string(UserOnboardingUpdate),
	string(UserOnboardingDelete),
	string(UserPreferenceRead),
	string(UserPreferenceUpdate),
	string(UserProfileImageCreate),
	string(UserProfileImageRead),
	string(UserProfileImageUpdate),
	string(UserProfileImageDelete),
	string(QueueRead),
	string(QueueUpdate),
	string(QueueJobCreate),
	string(QueueJobRead),
	string(QueueJobUpdate),
	string(QueueJobDelete),
	string(WorkflowCreate),
	string(WorkflowRead),
	string(WorkflowUpdate),
	string(WorkflowDelete),
	string(WorkflowLogs),
	string(AdminUserCreate),
	string(AdminUserRead),
	string(AdminUserUpdate),
	string(AdminUserDelete),
	string(AdminSessionRead),
	string(AdminAuthUnlinkAll),
}
