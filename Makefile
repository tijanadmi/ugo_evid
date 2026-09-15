DB_URL=postgresql://root:root@localhost:5433/tdi_evid?sslmode=disable
MIGRATE="C:/Users/tijana.dmitrovic/go/bin/migrate.exe"

postgres:
	docker run --name postgres16 --network tdi_evid-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=root -d postgres:16-alpine

createdb:
	docker exec -it postgres16 createdb --username=root --owner=root tdi_evid

dropdb:
	docker exec -it postgres16 dropdb tdi_evid

migrateup:
	$(MIGRATE) -path migrations -database "$(DB_URL)" -verbose up


migrateup1:
	$(MIGRATE) -path migrations -database "$(DB_URL)" -verbose up 1

migratedown:
	$(MIGRATE) -path migrations -database "$(DB_URL)" -verbose down

migratedown1:
	$(MIGRATE) -path migrations -database "$(DB_URL)" -verbose down 1

new_migration:
	$(MIGRATE) create -ext sql -dir migrations -seq $(name)
	
server:
	go run main.go

