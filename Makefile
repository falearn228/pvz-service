postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:17-alpine
startdb:
	docker start postgres
createdb:
	docker exec -it postgres createdb --username=root --owner=root pvz
dropdb:
	docker exec -it postgres dropdb pvz
migrateup:
	migrate -path migrations -database "postgresql://root:password@localhost:5432/pvz?sslmode=disable" -verbose up
migratedown:
	migrate -path migrations -database "postgresql://root:password@localhost:5432/pvz?sslmode=disable" -verbose down
test:
	go test -cover ./...
server:
	go run main.go
mock:
	mockgen -package mockdb -destination internal/mock/store.go pvz-service/internal/db/sqlc Store
####################################################################################################################################	
# Подготовка линтера
lint-prepare:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Проверка кода линтером
lint: lint-prepare
	golangci-lint run ./... --timeout=5m

# Исправление проблем линтером
lint-fix: lint-prepare
	golangci-lint run --fix ./...

# Проверка с выводом в JSON
lint-json: lint-prepare
	golangci-lint run ./... --out-format=json > lint-report.json

# Быстрая проверка только критических линтеров
lint-fast: lint-prepare
	golangci-lint run ./... --fast --disable-all -E errcheck,gosimple,govet

# Проверка с подробным выводом
lint-verbose: lint-prepare
	golangci-lint run ./... -v --timeout=5m	
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
.PHONY: postgres start createdb dropdb migrateup migratedown sqlc test server mock lint lint-fix lint-check load-test load-test-auth load-test-purchase load-test-transfer build up down logs ps clean
