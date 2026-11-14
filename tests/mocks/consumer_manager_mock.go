package mocks

import (
	"users-service/app/models/handlers"

	"github.com/stretchr/testify/mock"
)

type MockConsumerManager struct {
	mock.Mock
}

func (m *MockConsumerManager) Register(topic string, consumer *handlers.ConsumerWrapper) error {
	args := m.Called(topic, consumer)
	return args.Error(0)
}

func (m *MockConsumerManager) PingAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConsumerManager) GetConsumer(topic string) (*handlers.ConsumerWrapper, bool) {
	args := m.Called(topic)

	var r0 *handlers.ConsumerWrapper
	if args.Get(0) != nil {
		r0 = args.Get(0).(*handlers.ConsumerWrapper)
	}

	return r0, args.Bool(1)
}
