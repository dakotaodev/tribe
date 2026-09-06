.PHONY: api-run api-build api-test api-vet mobile-install mobile-start mobile-typecheck mobile-build test

api-run:
	cd api && go run ./cmd/server

api-build:
	cd api && mkdir -p bin && go build -o bin/tribe-api ./cmd/server

api-test:
	cd api && go test ./...

api-vet:
	cd api && go vet ./...

mobile-install:
	cd mobile && pnpm install

mobile-start:
	cd mobile && pnpm start

mobile-typecheck:
	cd mobile && pnpm typecheck

mobile-build:
	cd mobile && pnpm expo export --platform ios --output-dir dist

test: api-test api-vet mobile-typecheck
