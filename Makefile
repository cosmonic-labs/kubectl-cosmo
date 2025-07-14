.PHONY: build
build:
	go build -o bin/kubectl-cosmo cmd/main.go

.PHONY: deploy
deploy: build
	cp bin/kubectl-cosmo ~/.krew/bin/kubectl-cosmo