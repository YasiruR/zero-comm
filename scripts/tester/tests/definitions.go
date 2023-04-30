package tests

import "time"

type TestMode string

const (
	JoinLatency    TestMode = `join-latency`
	JoinThroughput TestMode = `join-throughput`
	PublishLatency TestMode = `publish-latency`
	MsgSize        TestMode = `message-size`
)

const (
	callbackEndpoint = `/callback`
)

var (
	numTests            int
	latncygrpSizes                    = []int{1, 2, 4, 8, 16}
	thrptGrpSizes                     = []int{4, 16}
	thrptJoinBatchSizes               = []int{4, 16}
	publishBatchSizes                 = []int{1, 10, 50, 100}
	agentId                           = 0
	agentPort                         = 6140
	pubPort                           = 6540
	testLatencyBuf      time.Duration = 2
	testMsgs                          = []string{
		`a`,
		`hello`,
		`sending hello didcomm world`,
		`this is a sample message for message size test with DIDcomm`,
		`this an extended sample message to check how larger messages impact the message size in bytes with respect to DIDComm packing protocols`,
		`DIDComm Messaging is a powerful way for people, institutions, and IoT things to interact via machine-readable messages, using features of decentralized identifiers (DIDs) as the basis of security and privacy. It works over any transport: HTTP, BlueTooth, SMTP, raw sockets, and sneakernet, for example. DIDComm Messaging is the first in a potential family of related specs; others could include DIDComm Streaming, DIDComm Multicast, and so forth. DIDComm is the common adjective for all of them, meaning that all will share DID mechanisms as the basis of security.`,
		`The purpose of DIDComm Messaging is to provide a secure, private communication methodology built atop the decentralized design of DIDs. It is the second half of this sentence, not the first, that makes DIDComm interesting. “Methodology” implies more than just a mechanism for individual messages, or even for a sequence of them. DIDComm Messaging defines how messages compose into the larger primitive of application-level protocols and workflows, while seamlessly retaining trust. “Built atop … DIDs” emphasizes DIDComm’s connection to the larger decentralized identity movement, with its many attendent virtues. Of course, robust mechanisms for secure communication already exist. However, most rely on key registries, identity providers, certificate authorities, browser or app vendors, or similar centralizations. Many are for unstructured rich chat only — or enable value-add behaviors through proprietary extensions. Many also assume a single transport, making it difficult to use the same solution for human and machine conversations, online and offline, simplex and duplex, across a broad set of modalities. And because these limitations constantly matter, composability is limited — every pairing of human and machine for new purposes requires a new endpoint, a new API, and new trust. Workflows that span these boundaries are rare and difficult.`,
		`The purpose of DIDComm Messaging is to provide a secure, private communication methodology built atop the decentralized design of DIDs. It is the second half of this sentence, not the first, that makes DIDComm interesting. “Methodology” implies more than just a mechanism for individual messages, or even for a sequence of them. DIDComm Messaging defines how messages compose into the larger primitive of application-level protocols and workflows, while seamlessly retaining trust. “Built atop … DIDs” emphasizes DIDComm’s connection to the larger decentralized identity movement, with its many attendent virtues. Of course, robust mechanisms for secure communication already exist. However, most rely on key registries, identity providers, certificate authorities, browser or app vendors, or similar centralizations. Many are for unstructured rich chat only — or enable value-add behaviors through proprietary extensions. Many also assume a single transport, making it difficult to use the same solution for human and machine conversations, online and offline, simplex and duplex, across a broad set of modalities. And because these limitations constantly matter, composability is limited — every pairing of human and machine for new purposes requires a new endpoint, a new API, and new trust. Workflows that span these boundaries are rare and difficult. All of these factors perpetuate an asymmetry between institutions and ordinary people. The former maintain certificates and always-connected servers, and publish APIs under terms and conditions they dictate; the latter suffer with usernames and passwords, poor interoperability, and a Hobson’s choice between privacy and convenience. DIDComm Messaging can fix these problems. Using DIDComm, individuals on semi-connected mobile devices become full peers of highly available web servers operated by IT experts. Registration is self-service, intermediaries require little trust, and terms and conditions can come from any party. DIDComm Messaging enables higher-order protocols that inherit its security, privacy, decentralization, and transport independence. Examples include exchanging verifiable credentials, creating and maintaining relationships, buying and selling, scheduling events, negotiating contracts, voting, presenting tickets for travel, applying to employers or schools or banks, arranging healthcare, and playing games. Like web services atop HTTP, the possibilities are endless; unlike web services atop HTTP, many parties can participate without being clients of a central server, and they can use a mixture of connectivity models and technologies. And these protocols are composable into higher-order workflows without constantly reinventing the way trust and identity transfer across boundaries.`,
	}
)
