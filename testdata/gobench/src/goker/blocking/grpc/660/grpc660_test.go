/*
 * Project: grpc-go
 * Issue or PR  : https://github.com/grpc/grpc-go/pull/660
 * Buggy version: db85417dd0de6cc6f583672c6175a7237e5b5dd2
 * fix commit-id: ceacfbcbc1514e4e677932fd55938ac455d182fb
 * Flaky: 100/100
 * Description:
 *   The parent function could return without draining the done channel.
 */
package grpc660

import (
	"math/rand"
	sched "sched"
	"testing"
)

type benchmarkClient struct {
	stop chan bool
}

func (bc *benchmarkClient) doCloseLoopUnary() {
	for {
		done := make(chan bool)
		go func() { // G2
			if rand.Intn(10) > 7 {
				sched.InstChBF(154618822657, done)
				done <- false
				sched.InstChAF(154618822657, done)
				return
			}
			sched.InstChBF(154618822658, done)
			done <- true
			sched.InstChAF(154618822658, done)
		}()
		select {
		case <-bc.stop:
			sched.InstChAF(154618822662, bc.stop)
			return
		case <-done:
			sched.InstChAF(

				///
				/// G1 						G2 				helper goroutine
				/// doCloseLoopUnary()
				///											bc.stop <- true
				/// <-bc.stop
				/// return
				/// 						done <-
				/// ----------------------G2 leak--------------------------
				///
				154618822663, done)
		}
	}
}

func TestGrpc660(t *testing.T) {
	bc := &benchmarkClient{
		stop: make(chan bool),
	}
	go bc.doCloseLoopUnary() // G1
	go func() {
		sched.InstChBF( // helper goroutine
			154618822661, bc.stop)
		bc.stop <- true
		sched.InstChAF(154618822661, bc.stop)
	}()
}
func TestGrpc660_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	bc := &benchmarkClient{
		stop: make(chan bool),
	}
	go bc.doCloseLoopUnary()
	go func() {
		sched.InstChBF(154618822661, bc.stop)
		bc.stop <- true
		sched.InstChAF(154618822661, bc.stop)
	}()
}
