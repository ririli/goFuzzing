package cockroach35501

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type MutableTableDescriptor struct {
	TableDescriptor
}

func (*MutableTableDescriptor) FindCheckByName(name string) {}

func NewMutableExistingTableDescriptor(tbl TableDescriptor) *MutableTableDescriptor {
	defer callstack.Trace(219043332097)()
	return &MutableTableDescriptor{TableDescriptor: tbl}
}

func validateCheckInTxn(tableDesc *MutableTableDescriptor, checkName *string) {
	defer callstack.Trace(219043332098)()
	tableDesc.FindCheckByName(*checkName)
}

type ConstraintToValidate struct {
	Name string
}

type SchemaChanger struct{}

type Descriptor struct{}

type TableDescriptor struct{}

func (*Descriptor) GetTable() *TableDescriptor {
	defer callstack.Trace(219043332099)()
	return &TableDescriptor{}
}

func GetTableDescFromID() *TableDescriptor {
	defer callstack.Trace(219043332100)()
	desc := &Descriptor{}
	return desc.GetTable()
}

type ImmutableTableDescriptor struct {
	TableDescriptor
}

func NewImmutableTableDescriptor(tbl TableDescriptor) *ImmutableTableDescriptor {
	defer callstack.Trace(219043332101)()
	return &ImmutableTableDescriptor{TableDescriptor: tbl}
}

func (desc *ImmutableTableDescriptor) MakeFirstMutationPublic() *MutableTableDescriptor {
	defer callstack.Trace(219043332102)()
	return NewMutableExistingTableDescriptor(desc.TableDescriptor)
}

func (*SchemaChanger) validateChecks(checks []ConstraintToValidate) {
	defer callstack.Trace(219043332103)()
	func() {
		defer callstack.Trace(219043332104)()
		tableDesc := GetTableDescFromID()
		desc := NewImmutableTableDescriptor(*tableDesc).MakeFirstMutationPublic()
		for _, c := range checks {
			go func() {
				defer callstack.Trace(219043332105)()
				validateCheckInTxn(desc, &c.Name)
			}()
		}
	}()
}

func (sc *SchemaChanger) runBackfill() {
	defer callstack.Trace(219043332106)()
	var checksToValidate []ConstraintToValidate
	for i := 0; i < 10; i++ {
		checksToValidate = append(checksToValidate, ConstraintToValidate{Name: "nil string"})
	}
	sc.validateChecks(checksToValidate)
}

func TestCockroach35501(t *testing.T) {
	defer callstack.Trace(219043332107)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(219043332108)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
func TestCockroach35501_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(219043332107)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(219043332108)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
