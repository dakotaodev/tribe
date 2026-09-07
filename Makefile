.PHONY: api-run api-build api-format api-test api-vet mobile-install mobile-start mobile-typecheck mobile-test mobile-build test

api-run:
	cd api && go run ./cmd/server

api-build:
	cd api && mkdir -p bin && go build -o bin/tribe-api ./cmd/server

api-format:
	@unformatted="$$(cd api && gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files must be formatted with gofmt:" >&2; \
		echo "$$unformatted" >&2; \
		exit 1; \
	fi

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

mobile-test:
	cd mobile && pnpm test

mobile-build:
	cd mobile && pnpm expo export --platform ios --output-dir dist

test: api-format api-vet api-test mobile-typecheck mobile-test
