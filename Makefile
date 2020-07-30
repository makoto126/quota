default: git

git:
	git add -A
	git commit -m "update"
	git push

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o out/quota *.go

deploy:

test: