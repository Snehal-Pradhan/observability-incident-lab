COMPOSE = docker compose

.PHONY: up down clean seed logs test

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

clean:
	$(COMPOSE) down -v

seed:
	$(COMPOSE) run --rm --entrypoint /usr/local/bin/seed api $(ARGS)

logs:
	$(COMPOSE) logs -f api

test:
	go test ./...
