package trace

import (
	"go/token"
	"math/rand"
	"sync"
	"time"
)

type routineInfo struct {
	id     int64
	start  time.Time
	end    time.Time
	pos    string
	finish bool
}

type chaninfo struct {
	decl token.Pos
	size int
}

type AllChanInfos struct {
	m sync.Map
}

func (info *AllChanInfos) Add(pos token.Pos, size int) {
	info.m.Store(pos, &chaninfo{pos, size})
}

func (info *AllChanInfos) Find(pos token.Pos) *chaninfo {
	if v, ok := info.m.Load(pos); ok {
		if vv, ok := v.(*chaninfo); ok {
			return vv
		}
	}
	return nil
}

var ChanInfos AllChanInfos

type AllInfos struct {
	m sync.Map
}

var allInfos AllInfos

func (info *AllInfos) add(pos string) int64 {
	id := rand.Int63()
	info.m.Store(id, &routineInfo{id, time.Now(), time.Now(), pos, false})
	return id
}

func (info *AllInfos) del(id int64) {
	if v, ok := info.m.Load(id); ok {
		if vv, ok := v.(*routineInfo); ok {
			vv.finish = true
			vv.end = time.Now()
		}
	}
}
