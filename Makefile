.PHONY: lint

lint:
	@fail=0; \
	(cd lib && golangci-lint run ./...) || fail=1; \
	(cd server && golangci-lint run ./...) || fail=1; \
	exit $$fail
