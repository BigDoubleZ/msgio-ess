package model

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var dbh *sql.DB

type ESSRec struct {
	ID      string
	Sender  string
	To      string
	Subject string
	Message string
}

func connect() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("PG_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func AddRecord(record ESSRec) error {
	db := connect()
	defer db.Close()

	const query = `insert into records ("id", "sender", "to", "subject", "message") 
		values ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, record.ID, record.Sender, record.To,
		record.Subject, record.Message)

	return err
}

func GetRecord(id string) (*ESSRec, error) {
	rec := new(ESSRec)

	db := connect()
	defer db.Close()

	const query = `select * from records where id = $1`

	row := db.QueryRow(query, id)
	err := row.Scan(rec.ID, rec.Sender, rec.To, rec.Subject, rec.Message)

	// switch err {
	// case sql.ErrNoRows:
	// 	fmt.Println("No rows")
	// 	return rec, nil

	// case nil:
	// 	return rec, nil

	// default:
	// 	return rec, err
	// }

	return rec, err
}

func SetRecordSent(id string, status bool) error {
	db := connect()
	defer db.Close()

	const query = `update records set sent_status = $2 where id = $1`

	_, err := db.Exec(query, id, status)
	return err
}
