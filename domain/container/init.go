package container

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/tryfix/log"
	"os"
)

type Args struct {
	Name     string
	Port     int
	PubPort  int
	ZmqBufMs int
	Mocker   bool
	MockPort int
	Verbose  bool
}

type Config struct {
	*Args
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

func (c *Container) Stop() error {
	if err := c.Server.Stop(); err != nil {
		return fmt.Errorf(`server shutdown failed - %v`, err)
	}

	if err := c.PubSub.Close(); err != nil {
		return fmt.Errorf(`group-agent shutdown failed - %v`, err)
	}

	c.Log.Info(`graceful shutdown of agent completed successfully`)
	os.Exit(0)
	return nil
}
