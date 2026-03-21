# Common tasks for tideproxy (see docs/specs/overview.md).
# Deploy defaults; override when invoking make, e.g. make deploy GCP_REGION=us-central1
# For gcpsetup: make gcpsetup GCP_PROJECT_ID=your-project-id

# Project ID from Google Cloud Console
GCP_PROJECT_ID = tides-proxy 

GCP_REGION ?= europe-west1

# Cloud Function name from Google Cloud Console
CF_NAME ?= tides-proxy

# Deployed HTTPS base (no trailing slash); override if URL changes.
EXAMPLE_REQUEST_CLOUD_BASE ?= https://europe-west1-tides-proxy.cloudfunctions.net/tides-proxy

# Must match the exported HTTP handler in package tideproxy (see entry.go).
CF_ENTRY_POINT ?= TidesProxy

.PHONY: gotest runlocalproxysvr deploy examplerequestcommandlocal examplerequestcommandcloud startlocalproxysvrandfirerequest gcpsetup

gotest:
	go test ./...

# Set active gcloud project and enable APIs used by Cloud Functions (2nd gen) deploy.
gcpsetup:
	@test -n "$(GCP_PROJECT_ID)" || { echo >&2 "GCP_PROJECT_ID must be set (e.g. make gcpsetup GCP_PROJECT_ID=my-project-id)"; exit 1; }
	gcloud config set project $(GCP_PROJECT_ID)
	gcloud services enable \
		artifactregistry.googleapis.com \
		cloudbuild.googleapis.com \
		cloudfunctions.googleapis.com \
		run.googleapis.com \
		logging.googleapis.com

runlocalproxysvr:
	@test -n "$$WORLDTIDES_API_KEY" || { echo >&2 "WORLDTIDES_API_KEY must be set"; exit 1; }
	go run ./cmd/tideproxy

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

# Requires a local server (e.g. make runlocalproxysvr in another terminal).
examplerequestcommandlocal:
	@curl -sS "http://127.0.0.1:8080/v1/tides?lat=50.351365&lon=-4.448837"

# Hits the deployed Cloud Function (see EXAMPLE_REQUEST_CLOUD_BASE).
examplerequestcommandcloud:
	@curl -sS "$(EXAMPLE_REQUEST_CLOUD_BASE)/v1/tides?lat=50.351365&lon=-4.448837"
