package domain

import (
	"github.com/tryfix/log"
)

type Config struct {
	Name             string
	Hostname         string
	Port             int
	InvEndpoint      string
	ExchangeEndpoint string
	Verbose          bool
	LogLevel         string
}

type Container struct {
	Cfg        *Config
	KS         KeyService
	Packer     Packer
	Tr         Transporter
	DS         DIDService
	OOB        OOBService
	Log        log.Logger
	InChanExch chan []byte
	InChanData chan []byte
	OutChan    chan string
}
