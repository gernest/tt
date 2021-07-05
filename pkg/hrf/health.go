package hrf

type Status string

const (
	Pass = Status("pass")
	Fail = Status("fail")
	Warn = Status("warn")
)

type Health struct {
	Status    Status `json:"status"`
	Version   string `json:"Version,omitempty"`
	ReleaseID string `json:"releaseId,omitempty"`
	ServiceID string `json:"serviceId,omitempty"`
}
