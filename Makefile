.PHONY: lint build-api build-poller docker-build-api docker-build-poller deploy-api

lint:
	@fail=0; \
	(cd lib && golangci-lint run ./...) || fail=1; \
	(cd server && golangci-lint run ./...) || fail=1; \
	exit $$fail

build-api:
	cd server && go build -o ../bin/api ./cmd/api

build-poller:
	cd server && go build -o ../bin/poller ./cmd/poller

docker-build-api:
	docker build --build-arg CMD=api -t dredge4us-api .

docker-build-poller:
	docker build --build-arg CMD=poller -t dredge4us-poller .

# First deploy:      make deploy-api
# Update an app:     make deploy-api APP_ID=<id from `doctl apps list`>
# Requires doctl authenticated (`doctl auth init`). Never put the real
# DATABASE_URL in .do/app.yaml — set it in the App Platform console after
# the first deploy. See .do/app.yaml for details.
deploy-api:
ifdef APP_ID
	doctl apps update $(APP_ID) --spec .do/app.yaml
else
	doctl apps create --spec .do/app.yaml
endif
