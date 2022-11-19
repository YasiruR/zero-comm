package domain

import (
	"github.com/tryfix/log"
)

type Args struct {
	Name    string
	Port    int
	Verbose bool
	PubPort int
}

type Config struct {
	Args
	Hostname         string
	InvEndpoint      string
	ExchangeEndpoint string
	PubEndpoint      string
	LogLevel         string
}

type Container struct {
	Cfg          *Config
	KS           KeyService
	Packer       Packer
	Tr           Transporter
	DS           DIDService
	OOB          OOBService
	Log          log.Logger
	InChan       chan Message
	SubChan      chan Message
	ConnDoneChan chan Connection
	Pub          Publisher
	Sub          Subscriber
	Prober       DIDCommService
	OutChan      chan string
}
