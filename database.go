package main

import (
	"database/sql"

	"github.com/ngageoint/seed-silo/models"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil { panic(err) }
	if db == nil { panic("db nil") }
	if err := db.Ping(); err != nil { panic(err) }

	models.CreateImageTable(db)
	models.CreateRegistryTable(db)

	return db
}