package peer

type State string

const (
	StateActive   State = "active"
	StateInactive State = "inactive"
	StateLeft     State = "left"
)
