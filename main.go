package main

import (
	"flag"
	"net"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/golang/glog"
)

type MatchSet struct {
	RR    string
	Type  string
	Value net.IP
}

func (m *MatchSet) hash() string {
	return m.RR + m.Type + m.Value.String()
}

type UpdateSet struct {
	RecordId string
	RR       string
	Type     string
	Value    string
}

func (u *UpdateSet) hash() string {
	return u.RR + u.Type + u.Value
}

// upstream host to sync
func upstream(host string, domains string) *[]MatchSet {
	realIPs, err := net.LookupIP(host)
	if err != nil || len(realIPs) < 1 {
		glog.Warningf("unable to found addr for %s", host)
		return nil
	}

	var recordTypes []string
	// can only handle one IP for each family
	var realV4IP, realV6IP net.IP
	for _, ip := range realIPs {
		glog.V(2).Infof("lookup %s %s", host, ip.String())
		if ip.To4() == nil {
			// v6
			realV6IP = ip
			recordTypes = append(recordTypes, "AAAA")
		} else {
			realV4IP = ip
			recordTypes = append(recordTypes, "A")
		}
	}

	// build matchset
	var matchSet []MatchSet
	rrs := strings.Split(domains, ",")
	for _, recordType := range recordTypes {
		var ip net.IP
		switch recordType {
		case "A":
			ip = realV4IP
		case "AAAA":
			ip = realV6IP
		default:
			continue
		}

		for _, rr := range rrs {
			matchSet = append(matchSet, MatchSet{
				RR:    rr,
				Type:  recordType,
				Value: ip,
			})
		}
	}
	return &matchSet
}

func sync(client *alidns.Client, domainName string, matchSet *[]MatchSet) {
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

	// 1) copy all
	createSet := make(map[string]UpdateSet, 2)
	for _, wantedRecord := range *matchSet {
		createSet[wantedRecord.hash()] = UpdateSet{
			RR:    wantedRecord.RR,
			Type:  wantedRecord.Type,
			Value: wantedRecord.Value.String(),
		}
	}

	// so this only check for update not add
	var updateSet []UpdateSet
	// match set A value RR
	for _, existedRecord := range response.DomainRecords.Record {
		for _, wantedRecord := range *matchSet {
			if existedRecord.Type != wantedRecord.Type {
				continue
			}
			if existedRecord.RR != wantedRecord.RR {
				continue
			}
			existedIP := net.ParseIP(existedRecord.Value)
			if existedIP == nil {
				continue
			}

			// found and existedRecord not changed
			if existedIP.Equal(wantedRecord.Value) {
				// remove from createSet
				delete(createSet, wantedRecord.hash())
				continue
			}
			toUpdate := UpdateSet{
				RecordId: existedRecord.RecordId,
				RR:       existedRecord.RR,
				Type:     existedRecord.Type,
				Value:    wantedRecord.Value.String(),
			}
			updateSet = append(updateSet, toUpdate)
			// remove from createSet
			delete(createSet, toUpdate.hash())
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

		_, err := client.UpdateDomainRecord(updateRequest)
		if err != nil {
			glog.Errorf("can not update record for aliyun dns, %s", err.Error())
			return
		}
	}

	for _, add := range createSet {
		addRequest := alidns.CreateAddDomainRecordRequest()
		addRequest.Scheme = "https"
		addRequest.AcceptFormat = "json"
		addRequest.Type = add.Type
		addRequest.RR = add.RR
		addRequest.Value = add.Value

		_, err := client.AddDomainRecord(addRequest)
		if err != nil {
			glog.Errorf("can not add record for aliyun dns, %s", err.Error())
			return
		}
	}
}

func main() {
	ak := flag.String("ak", "", "aliyun ak.")
	sk := flag.String("sk", "", "aliyun sk.")
	regionID := flag.String("region-id", "cn-hangzhou", "aliyun regionId default cn-hangzhou.")
	domainName := flag.String("domain-name", "", "domain registered in aliyun.")
	domainRR := flag.String("domain-rr", "", "comma separated list eg. @,www .")
	upstreamDomain := flag.String("upstream-domain", "", "upstream domain to sync.")

	flag.Parse()

	client, err := alidns.NewClientWithAccessKey(*regionID, *ak, *sk)
	if err != nil {
		glog.Fatal(err)
	}

	// sync immediately
	func() {
		matchSet := upstream(*upstreamDomain, *domainRR)
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
			matchSet := upstream(*upstreamDomain, *domainRR)
			if matchSet != nil {
				sync(client, *domainName, matchSet)
			}
		}
	}
}
