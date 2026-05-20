# Common tasks for tideproxy (see docs/specs/overview.md).
# Deploy defaults; override when invoking make, e.g. make deploy GCP_REGION=us-central1
#
# runlocalproxysvr and deploy require exported secrets (see README):
#   WORLDTIDES_API_KEY, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID
# Example: set -a && source .env && set +a

GCP_PROJECT_ID = tides-proxy
GCP_REGION ?= europe-west1
CF_NAME ?= tides-proxy
EXAMPLE_REQUEST_CLOUD_BASE ?= https://europe-west1-tides-proxy.cloudfunctions.net/tides-proxy
CF_ENTRY_POINT ?= TidesProxy

.PHONY: gotest runlocalproxysvr deploy examplerequestcommandlocal examplerequestcommandcloud gcpsetup

gotest:
	go test ./...

gcpsetup:
	@test -n "$(GCP_PROJECT_ID)" || { echo >&2 "GCP_PROJECT_ID must be set (e.g. make gcpsetup GCP_PROJECT_ID=my-project-id)"; exit 1; }
	gcloud config set project $(GCP_PROJECT_ID)
	gcloud services enable \
		artifactregistry.googleapis.com \
		cloudbuild.googleapis.com \
		cloudfunctions.googleapis.com \
		run.googleapis.com \
		logging.googleapis.com

runlocalproxysvr: require-runtime-secrets
	go run ./cmd/tideproxy

deploy: require-runtime-secrets
	gcloud functions deploy $(CF_NAME) \
		--gen2 \
		--runtime=go124 \
		--region=$(GCP_REGION) \
		--source=. \
		--entry-point=$(CF_ENTRY_POINT) \
		--trigger-http \
		--allow-unauthenticated \
		--set-env-vars=WORLDTIDES_API_KEY=$$WORLDTIDES_API_KEY,TELEGRAM_BOT_TOKEN=$$TELEGRAM_BOT_TOKEN,TELEGRAM_CHAT_ID=$$TELEGRAM_CHAT_ID

# Requires runlocalproxysvr in another terminal (after sourcing .env — see README).
examplerequestcommandlocal:
	@curl -sS "http://127.0.0.1:8080/v1/tides?lat=50.351365&lon=-4.448837"

examplerequestcommandcloud:
	@curl -sS "$(EXAMPLE_REQUEST_CLOUD_BASE)/v1/tides?lat=50.351365&lon=-4.448837"

.PHONY: require-runtime-secrets
require-runtime-secrets:
	@test -n "$$WORLDTIDES_API_KEY" || { echo >&2 "WORLDTIDES_API_KEY must be set (see README)"; exit 1; }
	@test -n "$$TELEGRAM_BOT_TOKEN" || { echo >&2 "TELEGRAM_BOT_TOKEN must be set (see README)"; exit 1; }
	@test -n "$$TELEGRAM_CHAT_ID" || { echo >&2 "TELEGRAM_CHAT_ID must be set (see README)"; exit 1; }
