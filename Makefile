.PHONY: up down test test-python test-go test-ts lint build clean

up:
	docker compose up --build -d

down:
	docker compose down

build:
	docker compose build

test: test-python test-go test-ts

test-python:
	cd services/metrics-collector && pip install -q -r requirements.txt && pytest -v

test-go:
	cd services/aggregator && go test -v ./...

test-ts:
	cd services/notifier && npm install --silent && npm test

lint: lint-python lint-go lint-ts

lint-python:
	cd services/metrics-collector && flake8 --max-line-length=120 app.py test_app.py

lint-go:
	cd services/aggregator && go vet ./...

lint-ts:
	cd services/notifier && npm install --silent && npx eslint src/ --ext .ts

logs:
	docker compose logs -f

health:
	@echo "metrics-collector:" && curl -s http://localhost:8080/health | python3 -m json.tool
	@echo "aggregator:" && curl -s http://localhost:8081/health | python3 -m json.tool
	@echo "notifier:" && curl -s http://localhost:8082/health | python3 -m json.tool

clean:
	docker compose down -v --rmi local
	rm -rf services/notifier/node_modules services/notifier/dist
	rm -f services/aggregator/aggregator
