package main

import (
	gw_uuid "github.com/generalworksinc/goutil/uuid"
	"github.com/oklog/ulid/v2"
	"log"
	"math/rand"
	"sort"
	"time"
)

func main() {
	ids1 := []string{}
	ids2 := []string{}
	for i := 1; i < 10000; i++ {
		sleepTime := int64(time.Millisecond) * int64(rand.Intn(20)+1)
		time.Sleep(time.Duration(sleepTime))
		println(i)
		id := gw_uuid.GetUlid()
		ids1 = append(ids1, id)
		ids2 = append(ids2, id)
	}

	sort.Strings(ids1)
	var id1Ulid ulid.ULID
	var id2Ulid ulid.ULID
	var err error
	for ind, id1 := range ids1 {
		id2 := ids2[ind]

		//id1UlidBefore := id1Ulid.String()
		id1Ulid, err = ulid.Parse(id1)
		if err != nil {
			log.Fatal("parse error 1.", id1)
		}
		id2UlidBefore := id2Ulid.String()
		id2Ulid, err = ulid.Parse(id2)
		if err != nil {
			log.Fatal("parse error 2.", id2)
		}

		if id1 != id2 {
			log.Fatal("ind, id1, id2, id1Time, id2Time", ind, id1, id2, ulid.Time(id1Ulid.Time()), ulid.Time(id2Ulid.Time()))
		}
		if id2UlidBefore > id2Ulid.String() {
			log.Fatal("before is Bigger!!!", ulid.Time(ulid.MustParse(id2UlidBefore).Time()), ulid.Time(id2Ulid.Time()))
		}
		if ids2[0] > id2Ulid.String() {
			log.Fatal("before is Bigger!!!", ulid.Time(ulid.MustParse(id2UlidBefore).Time()), ulid.Time(id2Ulid.Time()))
		}
		println(id1)
	}
	println("done! ok")
}
