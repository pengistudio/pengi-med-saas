package database

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/audit"
)

type DBExecute struct {
	ID      string                  `gorm:"primaryKey"`
	Execute func(db *gorm.DB) error `gorm:"-"`
}

type DBMap map[string]DBExecute

var GlobalDBMap = make(DBMap)

func NewDBMap() DBMap {
	return make(DBMap)
}

func ParseFileName(s string) (int, int) {
	// Eliminar prefijo "DB"
	parts := strings.Split(s[2:], "_")

	// Extraer y convertir la fecha
	dateStr := parts[0]
	day := dateStr[0:2]
	month := dateStr[2:4]
	year := dateStr[4:8]

	// Convertir la fecha a un formato cronológico (AAAAMMDD)
	date, _ := strconv.Atoi(year + month + day)

	// Convertir el ID
	id, _ := strconv.Atoi(parts[1])

	return date, id
}

func GetDB(c *gin.Context) *gorm.DB {
	return c.MustGet("db").(*gorm.DB)
}
func MigrateDB(db *gorm.DB, dst ...any) error {
	fmt.Println("Starting Database migration...")

	if err := db.AutoMigrate(dst...); err != nil {
		return fmt.Errorf("failed to auto-migrate Database: %w", err)
	}

	// Migrate standalone core models like AuditLog
	if err := db.AutoMigrate(&audit.AuditLog{}); err != nil {
		return fmt.Errorf("failed to auto-migrate AuditLog: %w", err)
	}

	if err := ExecuteAll(db); err != nil {
		return fmt.Errorf("Database migration failed: %w", err)
	}
	fmt.Println("Database migration completed successfully.")
	return nil
}

func ExecuteAll(db *gorm.DB) error {
	fmt.Println("🚀 Executing all Database migrations...")
	var DBList []DBExecute
	if err := db.Find(&DBList).Error; err != nil {
		return fmt.Errorf("failed to retrieve Database list: %w", err)
	}

	localDBMap := NewDBMap()
	for _, db := range DBList {
		localDBMap[db.ID] = db
	}

	if len(localDBMap) == len(GlobalDBMap) {
		fmt.Println("✅ All Database migrations already executed, skipping...")
		localDBMap = nil
		return nil
	}

	dbKeys := make([]string, 0, len(GlobalDBMap))
	for key := range GlobalDBMap {
		dbKeys = append(dbKeys, key)
	}

	sort.Slice(dbKeys, func(i, j int) bool {
		// Extraer la parte de la fecha y el ID
		dateI, idI := ParseFileName(dbKeys[i])
		dateJ, idJ := ParseFileName(dbKeys[j])

		// Comparar fechas primero
		if dateI != dateJ {
			return dateI < dateJ
		}
		// Si las fechas son iguales, comparar por ID
		return idI < idJ
	})

	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, key := range dbKeys {
			if _, exists := localDBMap[key]; exists {
				fmt.Printf("⏭️ Database Migration ID %s already executed, skipping...\n", key)
				continue
			}

			dvx := GlobalDBMap[key]
			if err := dvx.Execute(tx); err != nil {
				return fmt.Errorf("❌ failed to execute Database Migration %s: %w", key, err)
			}

			if err := tx.Create(&dvx).Error; err != nil {
				return fmt.Errorf("❌ failed to record Database Migration %s execution: %w", key, err)
			}
			fmt.Printf("✅ Database Migration %s executed successfully.\n", key)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("❌ transaction failed: %w", err)
	}

	fmt.Println("✅ All Database migrations executed successfully.")

	return nil
}
