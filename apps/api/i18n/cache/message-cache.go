package message_cache

import (
	message_models "pengi-med-saas/i18n/models"
	"sync"

	"gorm.io/gorm"
)

var (
	cache map[string]map[string]string // lang -> key -> value
	mutex sync.RWMutex
	once  sync.Once
)

func Init(db *gorm.DB) error {
	var err error
	once.Do(func() {
		cache = make(map[string]map[string]string)
		err = loadMessages(db)
	})
	return err
}

func loadMessages(db *gorm.DB) error {
	mutex.Lock()
	defer mutex.Unlock()

	var messages []message_models.Message
	if err := db.Find(&messages).Error; err != nil {
		return err
	}

	for _, msg := range messages {
		if cache[msg.Lang] == nil {
			cache[msg.Lang] = make(map[string]string)
		}
		cache[msg.Lang][msg.Key] = msg.Value
	}
	return nil
}

func Get(lang, key string) string {
	mutex.RLock()
	defer mutex.RUnlock()

	if cache[lang] != nil {
		if val, ok := cache[lang][key]; ok {
			return val
		}
	}

	// Fallback to Spanish or default key if needed
	if lang != "es" && cache["es"] != nil {
		if val, ok := cache["es"][key]; ok {
			return val
		}
	}

	return key
}

func Reload(db *gorm.DB) error {
	mutex.Lock()
	cache = make(map[string]map[string]string)
	mutex.Unlock()
	return loadMessages(db)
}
