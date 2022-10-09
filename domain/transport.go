package domain

type Transporter interface {
	Start()
	Send(data []byte, endpoint string) error
	Stop() error
}
