package configurator

const (
	messageRetentionMax     = 1209600
	messageRetentionMin     = 60
	messageRetentionDefault = messageRetentionMax

	visibilityTimeoutDefault = 60
	visibilityTimeoutMax     = 43200

	dlqMaxReceiveCountDefault = 7
	dlqSuffixDefault          = "_failures"

	fifoSuffix = ".fifo"
)
