package pubsub

type Agent struct{}

func (a *Agent) Join(topic, acceptor string) {
	// check if B is already connected with acceptor (A)
	// - only needs to check if ever connected (in agent's map), since disconnect is not implemented
	// if not, return
	// if connected, check if A's DID Doc has join-endpoint
	//  - save services in prober's peers map upon connection
	//  - open a func to get peer data
	// if join service is not present, return

	// call A's group-join/<topic> endpoint
	// save received group info in-memory (map with topics?)
	// create PUB and SUB sockets for msgs and statuses
	// for each pub in group info
	// - connect
	// - stores/updates pub in-memory

}

func (a *Agent) Connect(pub interface{}) {
	// if not already connected
	// - sets up DIDComm connection via inv
	// - B connects to pub via SUB for statuses and msgs
	// B sends agent subscribe msg to pub
	// B subscribes via zmq
}

func (a *Agent) initConnListener() {

}

func (a *Agent) initStatusListener() {
	// if status received,
	// - store/update in-memory
	// - if active
	//  - if B is a sub
	//    - if sender is a pub
	//      - if not already connected
	//		  - connect
	// - if not active, remove/disconnect
}

func (a *Agent) initMsgListener() {

}
