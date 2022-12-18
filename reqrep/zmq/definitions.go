package zmq

const (
	errTempUnavail = `resource temporarily unavailable`
)

const (
	successRes = `success`
	failedRes  = `failed`
)

// metadata is sent in a separate frame to preserve
// backward and forward compatibility
type metadata struct {
	Type string `json:"type"`
}
