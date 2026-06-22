package cockroach35501

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type MutableTableDescriptor struct {
	TableDescriptor
}

func (*MutableTableDescriptor) FindCheckByName(name string) {}

func NewMutableExistingTableDescriptor(tbl TableDescriptor) *MutableTableDescriptor {
	return &MutableTableDescriptor{TableDescriptor: tbl}
}

func validateCheckInTxn(tableDesc *MutableTableDescriptor, checkName *string) {
	tableDesc.FindCheckByName(*checkName)
}

type ConstraintToValidate struct {
	Name string
}

type SchemaChanger struct{}

type Descriptor struct{}

type TableDescriptor struct{}

func (*Descriptor) GetTable() *TableDescriptor {
	return &TableDescriptor{}
}

func GetTableDescFromID() *TableDescriptor {
	desc := &Descriptor{}
	return desc.GetTable()
}

type ImmutableTableDescriptor struct {
	TableDescriptor
}

func NewImmutableTableDescriptor(tbl TableDescriptor) *ImmutableTableDescriptor {
	return &ImmutableTableDescriptor{TableDescriptor: tbl}
}

func (desc *ImmutableTableDescriptor) MakeFirstMutationPublic() *MutableTableDescriptor {
	return NewMutableExistingTableDescriptor(desc.TableDescriptor)
}

func (*SchemaChanger) validateChecks(checks []ConstraintToValidate) {
	func() {
		tableDesc := GetTableDescFromID()
		desc := NewImmutableTableDescriptor(*tableDesc).MakeFirstMutationPublic()
		for _, c := range checks {
			go func() {
				goroutine.Enter(219043332097)
				defer goroutine.Exit(219043332097)
				func() {
					validateCheckInTxn(desc, &c.Name)
				}()
			}()
		}
	}()
}

func (sc *SchemaChanger) runBackfill() {
	var checksToValidate []ConstraintToValidate
	for i := 0; i < 10; i++ {
		checksToValidate = append(checksToValidate, ConstraintToValidate{Name: "nil string"})
	}
	sc.validateChecks(checksToValidate)
}

func TestCockroach35501(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		goroutine.Enter(219043332098)
		defer goroutine.Exit(219043332098)
		func() {
			defer wg.Done()
			sc := &SchemaChanger{}
			sc.runBackfill()
		}()
	}()
	wg.Wait()
}
func TestCockroach35501_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		goroutine.Enter(219043332098)
		defer goroutine.Exit(219043332098)
		func() {
			defer wg.Done()
			sc := &SchemaChanger{}
			sc.runBackfill()
		}()
	}()
	wg.Wait()
}
