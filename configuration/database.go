package configuration

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func NewDatabase() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:hbx@tcp(172.17.0.1:3306)/notitications-go")

	if err != nil {
		return nil, err
	}

	return db, nil
}
