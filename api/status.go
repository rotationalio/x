package api

const (
	serverStatusOK          = "ok"
	serverStatusNotReady    = "not ready"
	serverStatusUnhealthy   = "unhealthy"
	serverStatusMaintenance = "maintenance"
	serverStatusUnknown     = "unknown"
)

type ServerStatus uint8

const (
	StatusUnknown ServerStatus = iota
	StatusOK
	StatusNotReady
	StatusUnhealthy
	StatusMaintenance
)

func (s ServerStatus) String() string {
	switch s {
	case StatusOK:
		return serverStatusOK
	case StatusNotReady:
		return serverStatusNotReady
	case StatusUnhealthy:
		return serverStatusUnhealthy
	case StatusMaintenance:
		return serverStatusMaintenance
	default:
		return serverStatusUnknown
	}
}
