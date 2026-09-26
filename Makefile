.PHONY: api web generate

api:
	go run ./cmd/statecraft

web:
	cd web && npm run dev

generate:
	buf generate
