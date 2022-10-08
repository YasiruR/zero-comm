package domain

type Messenger interface {
	Connect()
	Sender
	Receiver
}

type Sender interface {
	Send(msg string)
}

type Receiver interface {
	Receive() (msg string)
}
