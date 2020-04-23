# msgio-ess
Simple e-mail sending service draft.

Experimental. Demo purposes only.

Build containers (from project root):

$ sudo docker build -t ess-sender -f sender/Dockerfile .
$ sudo docker build -t ess-acceptor -f acceptor/Dockerfile .

Configuration (env)

acceptor:
PG_DSN - PostgreSQL connection string (DSN)
MQ_URL - RabbitMQ connection URL
MQ_QUEUE - RabbitMQ queue name

sender:
(same as for acceptor, plus)
MAIL_SENDERADDRESS - sender e-mail address
MAIL_SERVERURL - SMTP server URL
MAIL_SERVERPORT - SMTP server port
MAIL_USER - SMTP auth user name
MAIL_PASSWORD - SMTP auth password

---

Not implemented:
- Tests
- DB/MQ reconnects
- API specs exposing
