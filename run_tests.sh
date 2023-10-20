clear
MONGODB_HOST=localhost MONGODB_PORT=27017 MONGODB_NAME=go_crud-test SERVER_PORT=8081 go test ./... -count=1 -race -coverprofile=c.out
go tool cover -html=c.out