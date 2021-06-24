package config

const (
	QosAtMostOnce byte = iota
	QosAtLeastOnce
	QosExactlyOnce
	QosFailure = 0x80
)

const (
	TypClient = iota
	TypServer
	TypTcp
	TypWs
)
