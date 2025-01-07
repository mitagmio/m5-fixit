# m5-fixit
go get -u github.com/swaggo/swag/cmd/swag
export PATH=$PATH:$(go env GOPATH)/bin
source ~/.bashrc
swag init -g cmd/main.go