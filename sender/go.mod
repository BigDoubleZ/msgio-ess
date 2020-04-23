module msgio-ess/sender

go 1.14

replace msgio-ess/model => ../model

require (
	github.com/lib/pq v1.4.0 // indirect
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71
	msgio-ess/model v0.0.0-00010101000000-000000000000
)
