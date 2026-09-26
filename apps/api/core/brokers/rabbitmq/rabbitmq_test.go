package rabbitmq

import (
	"errors"
	"testing"
	"time"

	"pengi-med-saas/core/logger"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func TestRetryCount(t *testing.T) {
	cases := []struct {
		headers amqp.Table
		want    int
	}{
		{nil, 0},
		{amqp.Table{}, 0},
		{amqp.Table{retryCountHeader: int32(2)}, 2},
		{amqp.Table{retryCountHeader: int64(3)}, 3},
		{amqp.Table{retryCountHeader: "junk"}, 0},
	}
	for _, tc := range cases {
		if got := retryCount(tc.headers); got != tc.want {
			t.Errorf("retryCount(%v) = %d, want %d", tc.headers, got, tc.want)
		}
	}
}

func TestQueueNames(t *testing.T) {
	if got := retryQueueName("invoice_tasks", 5*time.Minute); got != "invoice_tasks.retry.300s" {
		t.Errorf("retryQueueName = %q", got)
	}
	if got := deadLetterQueueName("invoice_tasks"); got != "invoice_tasks.dlq" {
		t.Errorf("deadLetterQueueName = %q", got)
	}
}

func TestRunHandlerRecoversPanic(t *testing.T) {
	logger.Log = zap.NewNop()

	err := runHandler("q", func([]byte) error { panic("boom") }, nil)
	if !errors.Is(err, errPanic) {
		t.Fatalf("err = %v, want errPanic", err)
	}

	sentinel := errors.New("handler failed")
	if err := runHandler("q", func([]byte) error { return sentinel }, nil); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want handler error passed through", err)
	}
}
