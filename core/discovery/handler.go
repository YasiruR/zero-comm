package discovery

//import (
//	"encoding/json"
//	"fmt"
//	"github.com/YasiruR/didcomm-prober/domain"
//	"github.com/YasiruR/didcomm-prober/domain/messages"
//	"github.com/YasiruR/didcomm-prober/domain/models"
//	"github.com/YasiruR/didcomm-prober/domain/services"
//	"github.com/google/uuid"
//	"strings"
//)
//
//type Discoverer struct {
//	tr        services.Transporter
//	queryChan chan models.Message
//	// feature values of key '*' are applied for all peers while further
//	// restrictions are listed in the same map under corresponding label
//	features []models.Feature
//}
//
//func NewDiscoverer() *Discoverer {
//	return &Discoverer{}
//}
//
//// Query creates the message as per RFC-0031 and requests from the
//// provided endpoint. As this may be used at any time by an agent,
//// didcomm support for messages is omitted.
//// see https://identity.foundation/didcomm-messaging/spec/#query-message-type
//func (d *Discoverer) Query(endpoint, query, comment string) error {
//	q := messages.QueryFeature{
//		Type:    messages.DiscoverFeatQuery,
//		Id:      uuid.New().String(),
//		Query:   query,
//		Comment: comment,
//	}
//
//	byts, err := json.Marshal(q)
//	if err != nil {
//		return fmt.Errorf(`marshalling query feature message failed - %v`, err)
//	}
//
//	msg, err := d.tr.Send(domain.MsgTypQuery, byts, endpoint)
//	if err != nil {
//		return fmt.Errorf(`sending query message failed - %v`, err)
//	}
//
//	// wait for response
//
//	// return supported features
//}
//
//func (d *Discoverer) Disclose(id, query string) messages.DiscloseFeature {
//	// filter wrt requester if required
//	var protocols []messages.Protocol
//	for _, f := range d.processQuery(query) {
//		protocols = append(protocols, messages.Protocol{PId: f.Id, Roles: f.Roles})
//	}
//
//	return messages.DiscloseFeature{
//		Type: messages.DiscoverFeatDisclose,
//		Thread: struct {
//			ThId string `json:"@thid"`
//		}{ThId: id},
//		Protocols: protocols,
//	}
//}
//
//// processQuery only performs a soft validation against the query as this
//// is only for the demonstration. Proper regex checks should be implemented
//// for a production release such that all edge-cases are covered.
//func (d *Discoverer) processQuery(query string) []models.Feature {
//	switch query {
//	case ``:
//		return d.features
//	case `*`:
//		return d.features
//	default:
//		// not the most accurate validation (since the inner-check may result in false positives)
//		if !strings.Contains(query, `https://didcomm.org/`) {
//			return nil
//		}
//
//		// if the query has a wild card
//		if string(query[len(query)-1]) == `*` {
//			var fs []models.Feature
//			for _, f := range d.features {
//				if strings.HasPrefix(f.Id, query[:len(query)-1]) {
//					fs = append(fs, f)
//				}
//			}
//			return fs
//		}
//
//		// if the query is specific to a feature
//		for _, f := range d.features {
//			if f.Id == query {
//				return []models.Feature{f}
//			}
//		}
//	}
//
//	return nil
//}
//
//func (d *Discoverer) listen() {
//	for {
//		req := <-d.queryChan
//
//	}
//}
