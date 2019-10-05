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

run with systemd

Raspberry Pi demo
```bash
tee /usr/lib/systemd/system/dns.service<<EOF
[Unit]
Description=dns

[Service]
User=pi
Type=simple
ExecStart=/usr/local/bin/dns  -v=2 --logtostderr --ak= --sk= -domain-name= --domain-rr=@,www --upstream-domain=

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable dns
systemctl start dns
```
