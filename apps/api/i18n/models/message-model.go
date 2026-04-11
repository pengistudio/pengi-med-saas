package message_models

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex:idx_key_lang;not null" json:"key"`
	Value string `gorm:"type:text;not null" json:"value"`
	Lang  string `gorm:"uniqueIndex:idx_key_lang;not null;default:es" json:"lang"` // ej: "es"
}

func NewMessage(key, value, lang string) *Message {
	return &Message{
		Key:   key,
		Value: value,
		Lang:  lang,
	}
}

func (m *Message) Save(db *gorm.DB) error {
	return db.Save(m).Error
}

func LoadMessagesFromFile(db *gorm.DB, filePath string, lang string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	for _, msg := range messages {
		msg.Lang = lang
		// Verificar si el mensaje ya existe
		var existingMsg Message
		result := db.Where("key = ? AND lang = ?", msg.Key, lang).Limit(1).Find(&existingMsg)
		if result.RowsAffected == 0 {
			// El mensaje no existe, lo guardamos
			if err := db.Create(&msg).Error; err != nil {
				return fmt.Errorf("error creating message %s: %w", msg.Key, err)
			}
		} else if result.Error != nil {
			// Otro tipo de error
			return fmt.Errorf("error checking message %s: %w", msg.Key, result.Error)
		} else {
			// El mensaje existe, verificar si el valor difiere
			if existingMsg.Value != msg.Value {
				// Actualizar el mensaje con el nuevo valor
				if err := db.Model(&existingMsg).Update("value", msg.Value).Error; err != nil {
					return fmt.Errorf("error updating message %s: %w", msg.Key, err)
				}
			}
		}
	}

	return nil
}

func LoadMessagesFromFS(db *gorm.DB, fs embed.FS, filename string, lang string) error {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading embedded file %s: %w", filename, err)
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	for _, msg := range messages {
		msg.Lang = lang
		var existingMsg Message
		result := db.Where("key = ? AND lang = ?", msg.Key, lang).Limit(1).Find(&existingMsg)
		if result.RowsAffected == 0 {
			if err := db.Create(&msg).Error; err != nil {
				return fmt.Errorf("error creating message %s: %w", msg.Key, err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("error checking message %s: %w", msg.Key, result.Error)
		} else if existingMsg.Value != msg.Value {
			if err := db.Model(&existingMsg).Update("value", msg.Value).Error; err != nil {
				return fmt.Errorf("error updating message %s: %w", msg.Key, err)
			}
		}
	}
	return nil
}
