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
	Hostname    string
	InvEndpoint string
	PubEndpoint string
	LogLevel    string
}

type Container struct {
	Cfg          *Config
	KeyManager   services.KeyManager
	Packer       services.Packer
	DidAgent     services.DIDUtils
	OOB          services.OutOfBand
	Connector    services.Connector
	Prober       services.Agent
	Client       services.Client
	Server       services.Server
	ConnDoneChan chan models.Connection
	OutChan      chan string
	Log          log.Logger
	PubSub       services.GroupAgent
}
