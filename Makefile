all: dev

DEV_FLAGS=--env-file=./dev.env \
	-f docker-compose.yml \
	-f docker-compose.dev.yml \

PROD_FLAGS=--env-file=./prod.env \
	-f docker-compose.yml \
	-f docker-compose.prod.yml \

.PHONY: dev
# Run the project in development mode.
dev:
	@echo "=== Running in development environement"
	@docker-compose $(DEV_FLAGS) up

.PHONY: prod
# Run the project in production mode.
prod:
	@echo "=== Running in production environement"
	@docker-compose $(PROD_FLAGS)	up --build

.PHONY: clean
# Cleanup any project trace on the host.
clean:
	@echo "=== Cleaning dev artifacts"
	@docker-compose $(DEV_FLAGS) down --volumes
	@echo "=== Cleaning prod artifacts"
	@docker-compose $(PROD_FLAGS) down --volumes
