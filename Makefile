# Common tasks for tideproxy (see docs/specs/overview.md).
# Deploy defaults; override when invoking make, e.g. make deploy GCP_REGION=us-central1

GCP_REGION ?= europe-west2
CF_NAME ?= tideproxy
# Any name works: the app registers a single unnamed HTTP handler; the framework
# maps FUNCTION_TARGET to that handler (see functions-framework-go registry).
CF_ENTRY_POINT ?= tideproxy

.PHONY: gotest runlocalproxysvr deploy examplerequestcommand startlocalproxysvrandfirerequest

gotest:
	go test ./...

runlocalproxysvr:
	@test -n "$$WORLDTIDES_API_KEY" || { echo >&2 "WORLDTIDES_API_KEY must be set"; exit 1; }
	go run .

deploy:
	@test -n "$$WORLDTIDES_API_KEY" || { echo >&2 "WORLDTIDES_API_KEY must be set (used for --set-env-vars)"; exit 1; }
	gcloud functions deploy $(CF_NAME) \
		--gen2 \
		--runtime=go124 \
		--region=$(GCP_REGION) \
		--source=. \
		--entry-point=$(CF_ENTRY_POINT) \
		--trigger-http \
		--allow-unauthenticated \
		--set-env-vars=WORLDTIDES_API_KEY=$$WORLDTIDES_API_KEY

examplerequestcommand:
	@echo 'curl -sS "http://127.0.0.1:8080/v1/tides?lat=51.5074&lon=-0.1278"'

startlocalproxysvrandfirerequest:
	@test -n "$$WORLDTIDES_API_KEY" || { echo >&2 "WORLDTIDES_API_KEY must be set"; exit 1; }
	@bash -euo pipefail -c '\
		go run . & pid=$$!; \
		trap "kill $$pid 2>/dev/null || true" EXIT; \
		for i in $$(seq 1 40); do \
			if curl -sfS "http://127.0.0.1:8080/v1/tides?lat=51.5074&lon=-0.1278"; then \
				exit 0; \
			fi; \
			sleep 0.25; \
		done; \
		echo >&2 "server did not become ready or request failed"; \
		exit 1'
