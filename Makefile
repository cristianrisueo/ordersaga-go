.PHONY: help up down logs check test

help: ## Lista los objetivos disponibles
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "%-8s %s\n", $$1, $$2}'

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
