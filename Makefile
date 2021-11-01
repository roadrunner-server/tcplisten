test:
	go test -race -v ./...

test_coverage:
	rm -rf coverage-ci
	mkdir ./coverage-ci
	go test -v -race -cover -coverpkg=./... -coverprofile=./coverage-ci/tcplisten.out -covermode=atomic ./...
	echo 'mode: atomic' > ./coverage-ci/summary.txt
	tail -q -n +2 ./coverage-ci/*.out >> ./coverage-ci/summary.txt
