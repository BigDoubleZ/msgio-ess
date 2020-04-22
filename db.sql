create table records
(
	id uuid not null
		constraint records_pk
			primary key,
	created_at timestamp(0) default now(),
	sender text,
	"to" text not null,
	subject text,
	message text not null,
	sent_status boolean default false
);

alter table records owner to postgres;

create unique index records_id_uindex
	on records (id);

create index records_created
	on records (created_at);

