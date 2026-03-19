package edge

import (
	"fmt"
)

type Publisher interface {
	Publish(action, key string)
}

type StdoutPublisher struct{}

func NewStdoutPublisher() StdoutPublisher {
	return StdoutPublisher{}
}

func (StdoutPublisher) Publish(action string, key string) {
	fmt.Printf("Publishing { \"action\": %s, \"key\":%s}\n", action, key)
}
