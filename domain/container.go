package domain

import (
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
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
	KS           services.KeyManager
	Packer       services.Packer
	Tr           services.Transporter
	DS           services.DID
	OOB          services.OOB
	Log          log.Logger
	InChan       chan models.Message
	SubChan      chan models.Message
	ConnDoneChan chan models.Connection
	Pub          services.Publisher
	Sub          services.Subscriber
	Prober       services.DIDComm
	OutChan      chan string
}
