package model

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type ESSRec struct {
	ID      string
	Sender  string
	To      string
	Subject string
	Message string
	Created string
	Sent    string
}

var db *sql.DB

func Setup() {
	db = connect()
}

func connect() *sql.DB {
	dbh, err := sql.Open("postgres", os.Getenv("PG_DSN"))
	if err != nil {
		log.Fatalf("[f] db: Error connecting: %s", err)
	}

	err = dbh.Ping()
	if err != nil {
		log.Fatalf("[f] db: Ping failed: %s", err)
	}

	log.Printf("[*] db: connected")
	return dbh
}

func AddRecord(record ESSRec) error {
	const query = `insert into records ("id", "sender", "to", "subject", "message") 
		values ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, record.ID, record.Sender, record.To,
		record.Subject, record.Message)

	return err
}

func GetRecord(id string) (*ESSRec, error) {
	const query = `select * from records where id = $1`

	rec := ESSRec{}

	row := db.QueryRow(query, id)
	err := row.Scan(&rec.ID, &rec.Created, &rec.Sender, &rec.To,
		&rec.Subject, &rec.Message, &rec.Sent)

	return &rec, err
}

func GetRecordList(limit int, offset int) ([]*ESSRec, error) {
	const query = `select * from records order by created_at, id asc limit $1 offset $2`

	var list []*ESSRec

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			return list, nil
		}
		return list, err
	}
	defer rows.Close()

	for rows.Next() {
		rec := ESSRec{}
		err := rows.Scan(&rec.ID, &rec.Created, &rec.Sender, &rec.To,
			&rec.Subject, &rec.Message, &rec.Sent)
		if err != nil {
			return list, err
		}
		list = append(list, &rec)
	}

	return list, nil
}

func RecordCount() (int, error) {
	const query = `select count(*) from records`
	var count int

	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}

func SetRecordSent(id string, status bool) error {
	const query = `update records set sent_status = $2 where id = $1`

	_, err := db.Exec(query, id, status)
	return err
}
