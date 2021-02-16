package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	_ "github.com/l1b0k/aliyun-ddns/addr/ipify"
	"github.com/l1b0k/aliyun-ddns/addr/types"
	"github.com/l1b0k/aliyun-ddns/version"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/golang/glog"
)

type MatchSet struct {
	DomainName string

	RR    string
	Type  string
	Value string
}

type UpdateSet struct {
	RecordId string
	RR       string
	Type     string
	Value    string
}

// upstream host to sync
func upstream(provider types.Interface, domains string) []MatchSet {
	realIPs, err := provider.Fetch()
	if err != nil {
		glog.Warningf("get addr failed %s", err.Error())
		return nil
	}
	if realIPs.IPv4 == nil && realIPs.IPv6 == nil {
		glog.Warningf("unable to found addr for %s")
		return nil
	}

	// build matchset
	var matchSet []MatchSet
	rrs := strings.Split(domains, ",")
	for _, rr := range rrs {
		if realIPs.IPv6 != nil {
			matchSet = append(matchSet, MatchSet{
				RR:    rr,
				Type:  "AAAA",
				Value: realIPs.IPv6.String(),
			})
		}
		if realIPs.IPv4 != nil {
			for _, rr := range rrs {
				matchSet = append(matchSet, MatchSet{
					RR:    rr,
					Type:  "A",
					Value: realIPs.IPv4.String(),
				})
			}
		}
	}

	return matchSet
}

func sync(client *alidns.Client, domainName string, matchSet []MatchSet) {
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.Scheme = "https"
	request.AcceptFormat = "json"
	request.DomainName = domainName

	response, err := client.DescribeDomainRecords(request)
	if err != nil {
		glog.Errorf("can not get record from aliyun dns %s", err.Error())
		return
	}

	// matchSet is what we expected
	// updateSet is what we will do update
	// createSet is (matchSet - updateSet) that is we will add

	// 1) add all to createSet
	createSet := make(map[MatchSet]struct{}, 2)
	for _, wantedRecord := range matchSet {
		createSet[wantedRecord] = struct{}{}
	}

	var updateSet []UpdateSet
	for _, existed := range response.DomainRecords.Record {
		for _, wanted := range matchSet {
			if existed.Type != wanted.Type {
				continue
			}
			if existed.RR != wanted.RR {
				continue
			}
			// check semantics
			existedIP := net.ParseIP(existed.Value)
			if existedIP != nil && existedIP.Equal(net.ParseIP(wanted.Value)) {
				delete(createSet, wanted)
				continue
			}

			toUpdate := UpdateSet{
				RecordId: existed.RecordId,
				RR:       existed.RR,
				Type:     existed.Type,
				Value:    wanted.Value,
			}
			updateSet = append(updateSet, toUpdate)
			// remove from createSet
			delete(createSet, wanted)
		}
	}

	for _, up := range updateSet {
		updateRequest := alidns.CreateUpdateDomainRecordRequest()
		updateRequest.Scheme = "https"
		updateRequest.AcceptFormat = "json"
		updateRequest.RecordId = up.RecordId
		updateRequest.Type = up.Type
		updateRequest.RR = up.RR
		updateRequest.Value = up.Value

		updateResp, err := client.UpdateDomainRecord(updateRequest)
		if err != nil {
			glog.Errorf("can not update record for aliyun dns, %s", err.Error())
			return
		}
		glog.Infof("update resolve record (%s %s %s) %s\n", domainName, up.RR, up.Value, updateResp.String())
	}

	for add := range createSet {
		addRequest := alidns.CreateAddDomainRecordRequest()
		addRequest.Scheme = "https"
		addRequest.AcceptFormat = "json"
		addRequest.Type = add.Type
		addRequest.RR = add.RR
		addRequest.Value = add.Value
		addRequest.DomainName = domainName

		createResp, err := client.AddDomainRecord(addRequest)
		if err != nil {
			glog.Errorf("can not add record for aliyun dns, %s", err.Error())
			return
		}
		glog.Infof("create resolve record (%s %s %s) %s\n", domainName, add.RR, add.Value, createResp.String())
	}
}

func init() {
	flag.Set("logtostderr", "true")
}

func main() {
	ak := flag.String("ak", "", "aliyun ak.")
	sk := flag.String("sk", "", "aliyun sk.")
	regionID := flag.String("region-id", "cn-hangzhou", "aliyun regionId default cn-hangzhou.")
	domainName := flag.String("domain-name", "", "domain registered in aliyun.")
	domainRR := flag.String("domain-rr", "", "comma separated list eg. @,www .")
	provider := flag.String("provider", "ipify", "provider to get public IP. eg. ipify")
	versionFlag := flag.Bool("version", false, "print version")

	flag.Parse()

	if *versionFlag {
		fmt.Println(version.Print())
		return
	}
	glog.Infof("version %s", version.Print())
	client, err := alidns.NewClientWithAccessKey(*regionID, *ak, *sk)
	if err != nil {
		glog.Fatal(err)
	}

	p, ok := types.Plugins[*provider]
	if !ok {
		glog.Fatal(fmt.Sprintf("can not found provider %s", *provider))
	}
	// sync immediately
	func() {
		matchSet := upstream(p(), *domainRR)
		if matchSet != nil {
			sync(client, *domainName, matchSet)
		}
	}()

	// aliyun has minimum TTL 600 (s)
	t := time.NewTicker(11 * time.Minute)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			matchSet := upstream(p(), *domainRR)
			if matchSet != nil {
				sync(client, *domainName, matchSet)
			}
		}
	}
}
