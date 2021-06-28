package configkey

const (
	CanonicalSnapStoreURL = "canonical.snap.store.url"
	LogLevel              = "log.level"
	DebugMode             = "debug"
	MinioAccessKey        = "minio.access.key"
	MinioSecretKey        = "minio.secret.key"
	MinioHost             = "minio.host"

	DatabaseUsername = "database.username"
	DatabaseDatabase = "database.database"
	DatabaseHost     = "database.host"
	DatabasePort     = "database.port"
	DatabaseSSLMode  = "database.sslmode"
	DatabaseTimezone = "database.timezone"
	DatabasePassword = "database.password"

	DashboardPort = "dashboard.port"
	LoginPort     = "login.port"

	StoreAPIURL = "store.api.url"

	OIDCClientId = "oidc.client.id"
	OIDCClientSecret = "oidc.client.secret"
	OIDCProviderURL = "oidc.provider.url"

	MacaroonDischargeKey = "macaroon.discharge.key"
	MacaroonRootKey = "macaroon.root.key"
	MacaroonRootId = "macaroon.root.id"
	MacaroonRootLocation = "macaroon.root.location"
	MacaroonThirdPartyCaveatId = "macaroon.thirdparty.caveat.id"
	MacaroonThirdPartyLocation = "macaroon.thirdparty.location"
)
