## aliyun-ddns

sync dns record via upstream domain.  

##  Concept

Smart router , NAS can provide free DDNS, but can only provide secondary domain.  
If you want to use your own domain ,you can't simply config on that.  
The whole point of DDNS is update the record when the ip is changed.  
So we can simply use the free DDNS and watch the change of ip.   
When the change is made ,we can update our domain record.   

## Build

```bash
go get -u github.com/l1b0k/aliyun-ddns
## arm
CGO_ENABLE=0 GOARCH=arm GOOS=linux go build -o dns main.go
```
## Usage

### run with docker

```bash
docker run --name ddns -d --restart=always l1b0k/aliyun-ddns --ak= --sk= --domain-name= --domain-rr=@,www
```

### run with systemd

Raspberry Pi demo
```bash
sudo cp dns.system /usr/lib/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable dns
sudo systemctl start dns
```
