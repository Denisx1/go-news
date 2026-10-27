.PHONY: help run build tidy init up down restart rebuild logs ps shell deploy df clean

COMPOSE ?= docker compose
SERVICE ?= gonews

ENV_FILE := .env
export

help: ## список команд
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-10s %s\n", $$1, $$2}'

# ---------- локально ----------
run: ## запуск без Docker
	go run ./cmd/gonews

build: ## сборка бинарника в bin/
	go build -o bin/gonews ./cmd/gonews

tidy: ## подтянуть зависимости
	go get github.com/joho/godotenv
	go mod tidy

# ---------- Docker (на сервере) ----------
init: ## первая настройка: .env, data/, prompts/
	@test -f .env || cp .env.example .env
	@mkdir -p data prompts
	@chown -R 1000:1000 data
	@echo "Заполните .env и положите старые state.json / news*.json в data/"

up: ## собрать и запустить в фоне
	$(COMPOSE) up -d --build

down: ## остановить и удалить контейнер
	$(COMPOSE) down

restart: ## перезапустить без пересборки (после правки .env или prompts/)
	$(COMPOSE) restart $(SERVICE)

rebuild: ## пересобрать образ с нуля
	$(COMPOSE) build --no-cache
	$(COMPOSE) up -d

logs: ## логи в реальном времени
	$(COMPOSE) logs -f --tail=100 $(SERVICE)

ps: ## статус контейнера
	$(COMPOSE) ps

shell: ## зайти внутрь контейнера
	$(COMPOSE) exec $(SERVICE) bash

deploy: ## git pull + пересборка + запуск
	git pull
	$(COMPOSE) up -d --build
	docker image prune -f

df: ## сколько места занимает Docker и свободно на диске
	@df -h /
	@docker system df

clean: ## освободить место: старые образы и кэш сборки
	docker image prune -f
	docker builder prune -f