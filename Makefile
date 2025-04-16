postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:17-alpine
startdb:
	docker start postgres
createdb:
	docker exec -it postgres createdb --username=root --owner=root pvz
dropdb:
	docker exec -it postgres dropdb pvz
migrateup:
	migrate -path migrations -database "postgresql://root:password@db:5432/pvz?sslmode=disable" -verbose up
migratedown:
	migrate -path migrations -database "postgresql://root:password@db:5432/pvz?sslmode=disable" -verbose down
test:
	go test -cover ./...
server:
	go run cmd/server/main.go
####################################################################################################################################		
# Сборка образов
build:
	docker-compose build

# Запуск всех сервисов
up:
	docker-compose up -d

# Остановка всех сервисов
down:
	docker-compose down

# Просмотр логов
logs:
	docker-compose logs -f

# Статус сервисов
ps:
	docker-compose ps

# Очистка всех данных
clean:
	docker-compose down -v
	docker system prune -f

# Перезапуск конкретного сервиса
restart-service:
	docker-compose restart $(service)
####################################################################################################################################
.PHONY: postgres start createdb dropdb migrateup migratedown sqlc test server build up down logs ps clean
