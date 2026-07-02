.PHONY: dev backend frontend test build

dev: backend frontend
	@echo "LifeOS running. Backend :3000, Frontend :5173"
	@echo "Open http://localhost:5173"

backend:
	cd backend && go run main.go &

frontend:
	cd frontend && npm run dev &

test:
	cd backend && go test ./... -v

build:
	cd frontend && npm run build
	cd backend && go build -o lifeos ./...
