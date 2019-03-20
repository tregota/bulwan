set CGO_ENABLED=0
set GOOS=linux
go build -a -o main .
docker build -t bulwan -f dockerfile .
docker save bulwan > bulwan.tar