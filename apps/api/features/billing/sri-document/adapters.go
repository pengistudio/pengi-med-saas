package sri_document

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"pengi-med-saas/core/brokers/rabbitmq"
	sri_services "pengi-med-saas/features/billing/sri/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewDefault wires the lifecycle to its production adapters: the sri-xml-signer
// service, tenant storage on local disk and RabbitMQ.
func NewDefault(db *gorm.DB, logger *zap.Logger) *Lifecycle {
	return New(db, logger, NewHTTPGateway(), DiskStorage{}, RabbitPublisher{}, sri_services.ResolveSriEnv())
}

// DiskStorage keeps signed XML under storage/tenants/<tenant>/<folder>/<key>.xml,
// relative to the working directory, next to each tenant's other files.
type DiskStorage struct{}

func (DiskStorage) ReadCertificate(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (DiskStorage) SaveSignedXML(tenantID uint, folder, accessKey, xml string) error {
	path := signedXMLPath(tenantID, folder, accessKey)
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(xml), 0644)
}

func (DiskStorage) RemoveSignedXML(tenantID uint, folder, accessKey string) error {
	if err := os.Remove(signedXMLPath(tenantID, folder, accessKey)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func signedXMLPath(tenantID uint, folder, accessKey string) string {
	return filepath.Join("storage", "tenants", fmt.Sprint(tenantID), folder, accessKey+".xml")
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
