package recycle

import (
	"testing"
	callstack "toolkit/pkg/callstack"
	"toolkit/pkg/operation"
)

func sum(n int) int {
	defer callstack.Trace(665719930881)()
	if n == 1 {
		return 1
	}
	return n * sum(n-1)
}
func sub(n int) int {
	defer callstack.Trace(665719930882)()
	n--
	return n
}
func TestA(t *testing.T) {
	defer callstack.Trace(665719930883)()

	go sum(3)
	go sub(4)
}
func TestA_1(t *testing.T) {
	callstack.ParseInput()
	operation.ParseInput()
	defer callstack.PrintTrees()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(665719930883)()
	go sum(3)
	go sub(4)

}
