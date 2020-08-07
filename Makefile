RUN_ARGS := ""
NEXT_TAG := $(shell git ls-remote --tags origin | grep -o 'refs/tags/[0-9]\+' | grep -o '[0-9]\+' | tr ' ' 'n' | sort -rn | head -n1 | xargs -I % echo '1+%' | bc)

.PHONY: test-up
test-up:
	docker container inspect db_test >/dev/null || (docker run -d --rm --name db_test -p 6666:5432 ggwp/db && sleep 15)

.PHONY: test-down
test-down:
	docker stop db_test

.PHONY: test
test:
	make test-up && go test -v -cover -covermode count ./... -run "$(RUN_ARGS)" | grep -v 'time=\"\|===\ RUN'

.PHONY: test-func-coverage
test-func:
	make test-up && go test ./... -coverprofile cover.out; go tool cover -func cover.out | egrep -i "$(RUN_ARGS)"; rm cover.out

.PHONY: checks
checks:
	which gosec || go get github.com/securego/gosec/cmd/gosec
	gosec ./...
	which buffy || go get github.com/spaceship-fspl/buffy
	buffy

.PHONY: release
release:
	$(shell test $(shell git rev-parse --abbrev-ref HEAD) = master)
	git tag $(NEXT_TAG)
	git push origin $(NEXT_TAG)
