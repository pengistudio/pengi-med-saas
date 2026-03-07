package database

import (
	"database/sql"
	"fmt"
	"log"
	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/config"

	_ "github.com/lib/pq" // driver PostgreSQL
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/*
Connect establishes a connection to the database using environment variables.
It returns a gorm.DB instance or an error if the connection fails.

It uses the PostgreSQL driver and requires the following environment variables:
- DB_HOST: The database host
- DB_PORT: The database port
- DB_USER: The database user
- DB_PASSWORD: The database password
- DB_NAME: The database name
If any of these variables are not set, it will return an error.
*/
func Connect() (*gorm.DB, error) {
	if err := EnsureDatabase(); err != nil {
		return nil, err
	}
	host := config.GetEnv("DB_HOST")
	port := config.GetEnv("DB_PORT")
	user := config.GetEnv("DB_USER")
	password := config.GetEnv("DB_PASSWORD")
	dbname := config.GetEnv("DB_NAME")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return nil, fmt.Errorf("missing required environment variables for database connection")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Register Audit Callbacks
	audit.RegisterCallbacks(db)

	return db, nil
}

/*
	EnsureDatabase checks if the specified database exists, and creates it if it does not.

It uses environment variables to get the database connection parameters:
- DB_HOST: The database host
- DB_PORT: The database port
- DB_USER: The database user
- DB_PASSWORD: The database password
- DB_NAME: The database name
If any of these variables are not set, it will return an error.
*/
func EnsureDatabase() error {
	host := config.GetEnv("DB_HOST")
	port := config.GetEnv("DB_PORT")
	user := config.GetEnv("DB_USER")
	password := config.GetEnv("DB_PASSWORD")
	dbname := config.GetEnv("DB_NAME")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return fmt.Errorf("missing required environment variables for database connection")
	}

	// 1️⃣ Conexión temporal sin especificar la base de datos
	rootDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable", host, port, user, password)
	rootDB, err := sql.Open("postgres", rootDSN)
	if err != nil {
		return fmt.Errorf("error connecting to postgres server: %w", err)
	}
	defer rootDB.Close()

	// 2️⃣ Verificar si la base ya existe
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = '%s');", dbname)
	if err := rootDB.QueryRow(query).Scan(&exists); err != nil {
		return fmt.Errorf("error checking database existence: %w", err)
	}

	// 3️⃣ Crear si no existe
	if !exists {
		_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbname))
		if err != nil {
			return fmt.Errorf("error creating database %s: %w", dbname, err)
		}
		log.Printf("✅ Database %s created successfully\n", dbname)
	} else {
		log.Printf("ℹ️ Database %s already exists\n", dbname)
	}

	return nil
}
