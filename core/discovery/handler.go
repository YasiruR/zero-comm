package discovery

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/google/uuid"
	"github.com/tryfix/log"
	"strings"
)

type Discoverer struct {
	queryChan chan models.Message
	client    services.Client
	server    services.Server
	// feature values of key '*' are applied for all peers while further
	// restrictions are listed in the same map under corresponding label
	features []models.Feature
	log      log.Logger
}

func NewDiscoverer(c *domain.Container) *Discoverer {
	d := &Discoverer{
		queryChan: make(chan models.Message),
		client:    c.Client,
		server:    c.Server,
		features: []models.Feature{
			{Id: `1`, Roles: []string{`tester`}},
			{Id: `22`, Roles: []string{`huuuuu`}},
			{Id: `245`, Roles: []string{`another`}},
		},
		log: c.Log,
	}

	d.server.AddHandler(domain.MsgTypQuery, d.queryChan, false)
	go d.listen()
	return d
}

func (d *Discoverer) listen() {
	for {
		msg := <-d.queryChan
		var qm messages.QueryFeature
		if err := json.Unmarshal(msg.Data, &qm); err != nil {
			d.log.Error(fmt.Sprintf(`invalid message received as a discovery query (%s) - %v`, string(msg.Data), err))
			continue
		}

		dm := d.Disclose(qm.Id, qm.Query)
		byts, err := json.Marshal(dm)
		if err != nil {
			d.log.Error(fmt.Sprintf(`marshalling disclose response failed - %v`, err))
			continue
		}

		// sending response to the reply channel
		msg.Reply <- byts
	}
}

// Query creates the message as per RFC-0031 and requests from the
// provided endpoint. As this may be used at any time by an agent,
// didcomm support for messages is omitted.
// see https://identity.foundation/didcomm-messaging/spec/#query-message-type
func (d *Discoverer) Query(endpoint, query, comment string) (fs []models.Feature, err error) {
	q := messages.QueryFeature{
		Type:    messages.DiscoverFeatQuery,
		Id:      uuid.New().String(),
		Query:   query,
		Comment: comment,
	}

	byts, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf(`marshalling query feature message failed - %v`, err)
	}

	res, err := d.client.Send(domain.MsgTypQuery, byts, endpoint)
	if err != nil {
		return nil, fmt.Errorf(`sending query message failed - %v`, err)
	}

	var dm messages.DiscloseFeature
	if err = json.Unmarshal([]byte(res), &dm); err != nil {
		return nil, fmt.Errorf(`unmarshalling disclose response failed - %v`, err)
	}

	return dm.Features, nil
}

func (d *Discoverer) Disclose(id, query string) messages.DiscloseFeature {
	// filter wrt requester if required
	return messages.DiscloseFeature{
		Type: messages.DiscoverFeatDisclose,
		Thread: struct {
			ThId string `json:"@thid"`
		}{ThId: id},
		Features: d.processQuery(query),
	}
}

// processQuery only performs a soft validation against the query as this
// is only for the demonstration. Proper regex checks should be implemented
// for a production release such that all edge-cases are covered.
func (d *Discoverer) processQuery(query string) []models.Feature {
	switch query {
	case ``:
		return d.features
	case `*`:
		return d.features
	default:
		// not the most accurate validation (since the inner-check may result in false positives)
		// eg: `abchttps://didcomm.org/`
		if !strings.Contains(query, `https://didcomm.org/`) {
			return nil
		}

		// if the query has a wild card at the end
		// this neglects the cases where wild card occurs in the middle
		if string(query[len(query)-1]) == `*` {
			var fs []models.Feature
			for _, f := range d.features {
				if strings.HasPrefix(f.Id, query[:len(query)-1]) {
					fs = append(fs, f)
				}
			}
			return fs
		}

		// if the query is specific to a feature
		for _, f := range d.features {
			if f.Id == query {
				return []models.Feature{f}
			}
		}
	}

	return nil
}
