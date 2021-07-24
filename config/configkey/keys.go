package configkey

const (
	CanonicalSnapStoreURL = "canonical.snap.store.url"
	LogLevel              = "log.level"
	DebugMode             = "debug"
	MinioAccessKey        = "minio.access.key"
	MinioSecretKey        = "minio.secret.key"
	MinioHost             = "minio.host"
	MinioSecure = "minio.secure"

	DatabaseUsername = "database.username"
	DatabaseDatabase = "database.database"
	DatabaseHost     = "database.host"
	DatabasePort     = "database.port"
	DatabaseSSLMode  = "database.sslmode"
	DatabaseTimezone = "database.timezone"
	DatabasePassword = "database.password"

	DashboardURL = "dashboard.url"
	StoreURL = "store.url"
	LoginURL = "login.url"

	DashboardPort = "dashboard.port"
	LoginPort     = "login.port"
	AdminDPort = "admind.port"
	AdminDURL = "admind.url"

	StoreAPIURL = "store.api.url"
	StoreInitializationConfigPath = "store.initialization.config.path"

	OIDCClientId = "oidc.client.id"
	OIDCClientSecret = "oidc.client.secret"
	OIDCProviderURL = "oidc.provider.url"

	MacaroonDischargeKey = "macaroon.discharge.key"
	MacaroonRootKey = "macaroon.root.key"
	MacaroonRootId = "macaroon.root.id"
	MacaroonRootLocation = "macaroon.root.location"
	MacaroonThirdPartyCaveatId = "macaroon.thirdparty.caveat.id"
	MacaroonThirdPartyLocation = "macaroon.thirdparty.location"

	RootAuthority = "root.authority"

	AdminCLILoginPort = "admin.login.port"
)
