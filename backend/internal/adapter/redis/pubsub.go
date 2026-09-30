package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func PublishMessage(ctx context.Context, rdt *redis.Client, channelName string, payload string) error {
	if err := rdt.Publish(ctx, channelName, payload).Err(); err != nil {
		return fmt.Errorf("publish message to %s: %w", channelName, err)
	}

	return nil
}

func SubscribeChannel(ctx context.Context, rdt *redis.Client, channelName string) *redis.PubSub {
	return rdt.Subscribe(ctx, channelName)
}
