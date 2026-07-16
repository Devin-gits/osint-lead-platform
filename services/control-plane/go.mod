module github.com/Moyeil-73/osint-lead-platform/services/control-plane

go 1.22.5

require (
	github.com/Moyeil-73/osint-lead-platform/modules/domain-intel v0.0.0-00010101000000-000000000000
	github.com/Moyeil-73/osint-lead-platform/modules/email-validate v0.0.0
	github.com/Moyeil-73/osint-lead-platform/modules/phone-validate v0.0.0
	github.com/jackc/pgx/v5 v5.6.0
)

require (
	github.com/AfterShip/email-verifier v1.4.1 // indirect
	github.com/hbollon/go-edlib v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/likexian/whois v1.15.6 // indirect
	github.com/nyaruka/phonenumbers v1.5.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/exp v0.0.0-20240525044651-4c93da0ed11d // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace (
	github.com/Moyeil-73/osint-lead-platform/modules/domain-intel => ../../modules/domain-intel
	github.com/Moyeil-73/osint-lead-platform/modules/email-validate => ../../modules/email-validate
	github.com/Moyeil-73/osint-lead-platform/modules/phone-validate => ../../modules/phone-validate
)
