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
	defer callstack.Trace(545460846593)()
	return &MutableTableDescriptor{TableDescriptor: tbl}
}

func validateCheckInTxn(tableDesc *MutableTableDescriptor, checkName *string) {
	defer callstack.Trace(545460846594)()
	tableDesc.FindCheckByName(*checkName)
}

type ConstraintToValidate struct {
	Name string
}

type SchemaChanger struct{}

type Descriptor struct{}

type TableDescriptor struct{}

func (*Descriptor) GetTable() *TableDescriptor {
	defer callstack.Trace(545460846595)()
	return &TableDescriptor{}
}

func GetTableDescFromID() *TableDescriptor {
	defer callstack.Trace(545460846596)()
	desc := &Descriptor{}
	return desc.GetTable()
}

type ImmutableTableDescriptor struct {
	TableDescriptor
}

func NewImmutableTableDescriptor(tbl TableDescriptor) *ImmutableTableDescriptor {
	defer callstack.Trace(545460846597)()
	return &ImmutableTableDescriptor{TableDescriptor: tbl}
}

func (desc *ImmutableTableDescriptor) MakeFirstMutationPublic() *MutableTableDescriptor {
	defer callstack.Trace(545460846598)()
	return NewMutableExistingTableDescriptor(desc.TableDescriptor)
}

func (*SchemaChanger) validateChecks(checks []ConstraintToValidate) {
	defer callstack.Trace(545460846599)()
	func() {
		defer callstack.Trace(545460846600)()
		tableDesc := GetTableDescFromID()
		desc := NewImmutableTableDescriptor(*tableDesc).MakeFirstMutationPublic()
		for _, c := range checks {
			go func() {
				defer callstack.Trace(545460846601)()
				validateCheckInTxn(desc, &c.Name)
			}()
		}
	}()
}

func (sc *SchemaChanger) runBackfill() {
	defer callstack.Trace(545460846602)()
	var checksToValidate []ConstraintToValidate
	for i := 0; i < 10; i++ {
		checksToValidate = append(checksToValidate, ConstraintToValidate{Name: "nil string"})
	}
	sc.validateChecks(checksToValidate)
}

func TestCockroach35501(t *testing.T) {
	defer callstack.Trace(545460846603)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(545460846604)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
func TestCockroach35501_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(545460846603)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(545460846604)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
