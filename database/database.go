package database

import (
	"database/sql"
	"os"

	"github.com/ngageoint/seed-silo/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/lib/pq"
)

var data *sql.DB

var dbType string

func CreateSqliteDB(filepath string) {
	os.Remove(filepath)
}

func InitSqliteDB(filepath, admin, password string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil { panic(err) }
	if db == nil { panic("db nil") }
	if err := db.Ping(); err != nil { panic(err) }

	data = db
	dbType = "sqlite"

	models.CreateImageTable(db, dbType)
	models.CreateRegistryTable(db, dbType)
	models.CreateUser(db, dbType, admin, password)
	models.CreateJobTable(db, dbType)
	models.CreateJobVersionTable(db, dbType)

	return db
}

func CreatePostgresDB(url, name string) {
	connection, _ := pq.ParseURL(url)
	db, err := sql.Open("postgres", connection)
	if err != nil { panic(err) }
	if db == nil { panic("db nil") }
	if err := db.Ping(); err != nil { panic(err) }

	db.Exec("DROP DATABASE IF EXISTS " + name)
	db.Exec("CREATE DATABASE " + name)
}

func RemovePostgresDB(url, name string) {
	connection, _ := pq.ParseURL(url)
	db, err := sql.Open("postgres", connection)
	if err != nil { panic(err) }
	if db == nil { panic("db nil") }
	if err := db.Ping(); err != nil { panic(err) }

	db.Exec("DROP DATABASE IF EXISTS " + name)
}

func InitPostgresDB(url, admin, password string) *sql.DB {
    connection, _ := pq.ParseURL(url)
    connection = connection + " search_path=silo"
    db, err := sql.Open("postgres", connection)
    db.Exec("CREATE SCHEMA silo")

    if err != nil { panic(err) }
    if db == nil { panic("db nil") }
    if err := db.Ping(); err != nil { panic(err) }

	data = db
	dbType = "postgres"

	models.CreateRegistryTable(db, dbType)
	models.CreateJobTable(db, dbType)
	models.CreateJobVersionTable(db, dbType)
	models.CreateImageTable(db, dbType)
	models.CreateUser(db, dbType, admin, password)

	return db
}

func GetDB() *sql.DB {
	return data
}

func GetDbType() string {
	return dbType
}
