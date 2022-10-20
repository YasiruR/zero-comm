package domain

const (
	InvitationEndpoint = `/invitation/`
	ExchangeEndpoint   = `/did-exchange/`
)

type Config struct {
	Name     string
	Port     int
	Hostname string
}
