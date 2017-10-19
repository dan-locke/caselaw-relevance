package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type dbConfig struct {

	user string

	pass string

	collection string

	host string

	port string

}

func initDatabase(conf dbConfig) (*sql.DB, error) {
	return sql.Open("postgres",
		fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
			conf.user, conf.pass, conf.collection, conf.host, conf.port))
}
