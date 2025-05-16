package mocks

import (
	"context"

	"github.com/alipio/awsconfigurator-go/configurator"
	"github.com/stretchr/testify/mock"
)

type AwsProvider struct {
	mock.Mock
}

func (p *AwsProvider) CreateTopic(ctx context.Context, topic configurator.Topic) (string, error) {
	args := p.Called(ctx, topic)
	return args.String(0), args.Error(1)
}

func (p *AwsProvider) CreateQueue(ctx context.Context, queue configurator.Queue) (string, error) {
	args := p.Called(ctx, queue)
	return args.String(0), args.Error(1)
}

func (p *AwsProvider) Subscribe(ctx context.Context, queue configurator.Queue, topics []configurator.Topic) error {
	args := p.Called(ctx, queue, topics)
	return args.Error(1)
}
