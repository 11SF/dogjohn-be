# TDG Configuration

## Project Information
- Language: Go 1.25
- Framework: Gin (github.com/gin-gonic/gin)
- Test Framework: go test + testify + go.uber.org/mock

## Build Command
go build ./...

## Test Command
go test -v -race -covermode=atomic -buildvcs -coverpkg=./... ./...

## Single Test Command
go test -v -run TestFunctionName ./path/to/package

## Coverage Command
go test -v -race -buildvcs -coverprofile=./coverage.out ./... && go tool cover -html=./coverage.out

## Test File Patterns
- Test files: *_test.go
- Test directory: alongside source files (Go convention)
