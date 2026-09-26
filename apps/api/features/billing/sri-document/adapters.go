package sri_document

import (
	"errors"

	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	sri_services "pengi-med-saas/features/billing/sri/services"
	billing_templates "pengi-med-saas/features/billing/templates"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewDefault wires the lifecycle to its production adapters: the sri-xml-signer
// service, tenant files on local disk, Gotenberg and RabbitMQ.
func NewDefault(db *gorm.DB, logger *zap.Logger) *Lifecycle {
	files := tenantfiles.Disk(tenantfiles.DefaultRoot)
	docs := Documents{
		Files:    files,
		Renderer: pdfrender.New(files, pdfrender.Gotenberg(), billing_templates.FS),
	}
	return New(db, logger, NewHTTPGateway(), RabbitPublisher{}, docs, sri_services.ResolveSriEnv())
}

// RabbitPublisher publishes on the process-wide RabbitMQ publish channel.
type RabbitPublisher struct{}

var errBrokerDisconnected = errors.New("rabbitmq disconnected")

func (RabbitPublisher) Publish(queue string, body []byte) error {
	ch := rabbitmq.PublishChannel()
	if ch == nil {
		return errBrokerDisconnected
	}
	return rabbitmq.PublishMessage(ch, queue, body)
}
