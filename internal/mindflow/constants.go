package mindflow

const (
	csrfKey = "csrf"

	eventQueueCapacity   = 16
	identifierBytes      = 32
	jsonRequestBytes     = 17 << 20
	maxRateWindows       = 10000
	maxRequestsPerMinute = 20
)
