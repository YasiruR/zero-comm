package domain

import (
	"github.com/tryfix/log"
)

const (
	InvitationEndpoint = `/invitation/`
	ExchangeEndpoint   = `/did-exchange/`
	MessageEndpoint    = `/didcomm-message/` // todo use this
)

type Config struct {
	Name             string
	Hostname         string
	Port             int
	InvEndpoint      string
	ExchangeEndpoint string
	MessageEndpoint  string
	Verbose          bool
	LogLevel         string
}

type Container struct {
	Cfg     *Config
	KS      KeyService
	Packer  Packer
	Tr      Transporter
	DS      DIDService
	OOB     OOBService
	Logger  log.Logger
	InChan  chan []byte
	OutChan chan string
}
