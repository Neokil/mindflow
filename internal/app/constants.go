package app

const (
	csrfKey               = "csrf"
	eventHeartbeatSeconds = 15
	eventQueueCapacity    = 16
	identifierBytes       = 32
	jsonRequestBytes      = 17 << 20
	maxRateWindows        = 10000
	maxRequestsPerMinute  = 20
	multipartRequestBytes = 21 << 20
)

const (
	maxUploadBytes       = 20 << 20
	multipartMemoryBytes = 1 << 20
)

const sessionLifetimeSeconds = 30 * 86400
