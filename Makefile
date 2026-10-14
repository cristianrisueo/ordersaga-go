.PHONY: help up down logs check test migration

# Versión de goose de go.mod, para que la CLI coincida con la librería
GOOSE_VERSION = $(shell go list -m -f '{{.Version}}' github.com/pressly/goose/v3)

help: ## Lista los objetivos disponibles
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "%-10s %s\n", $$1, $$2}'

up: ## Levanta los contenedores y espera a que estén operativos
	docker compose up -d --build --wait

down: ## Para los contenedores y elimina volúmenes
	docker compose down -v

logs: ## Sigue los logs de los contenedores
	docker compose logs -f

check: ## Formatea y pasa golangci-lint
	golangci-lint fmt
	golangci-lint run

test: ## Tests con el detector de carreras
	go test -race ./...

migration: ## Crea una migración: make migration service=<servicio> name=<descripción>
	@if [ -z "$(service)" ] || [ -z "$(name)" ]; then \
		echo "uso: make migration service=<servicio> name=<descripción>"; \
		exit 1; \
	fi
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir migrations/$(service) -s create $(name) sql
