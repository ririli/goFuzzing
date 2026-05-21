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
	//defer callstack.Trace(515396075521)()
	defer callstack.Trace(515396075521)()

	return &MutableTableDescriptor{TableDescriptor: tbl}
}

func validateCheckInTxn(tableDesc *MutableTableDescriptor, checkName *string) {
	//	defer callstack.Trace(515396075522)()
	defer callstack.Trace(515396075522)()

	tableDesc.FindCheckByName(*checkName)
}

type ConstraintToValidate struct {
	Name string
}

type SchemaChanger struct{}

type Descriptor struct{}

type TableDescriptor struct{}

func (*Descriptor) GetTable() *TableDescriptor {
	//defer callstack.Trace(515396075523)()
	defer callstack.Trace(515396075523)()

	return &TableDescriptor{}
}

func GetTableDescFromID() *TableDescriptor {
	//defer callstack.Trace(515396075524)()
	defer callstack.Trace(515396075524)()

	desc := &Descriptor{}
	return desc.GetTable()
}

type ImmutableTableDescriptor struct {
	TableDescriptor
}

func NewImmutableTableDescriptor(tbl TableDescriptor) *ImmutableTableDescriptor {
	//defer callstack.Trace(515396075525)()
	defer callstack.Trace(515396075525)()

	return &ImmutableTableDescriptor{TableDescriptor: tbl}
}

func (desc *ImmutableTableDescriptor) MakeFirstMutationPublic() *MutableTableDescriptor {
	//defer callstack.Trace(515396075526)()
	defer callstack.Trace(515396075526)()

	return NewMutableExistingTableDescriptor(desc.TableDescriptor)
}

func (*SchemaChanger) validateChecks(checks []ConstraintToValidate) {
	//defer callstack.Trace(515396075527)()
	defer callstack.Trace(515396075527)()

	func() {
		//defer callstack.Trace(515396075528)()
		defer callstack.Trace(515396075528)()
		tableDesc := GetTableDescFromID()
		desc := NewImmutableTableDescriptor(*tableDesc).MakeFirstMutationPublic()
		for _, c := range checks {
			go func() {
				//defer callstack.Trace(515396075529)()
				defer callstack.Trace(515396075529)()
				validateCheckInTxn(desc, &c.Name)
			}()
		}
	}()
}

func (sc *SchemaChanger) runBackfill() {
	//defer callstack.Trace(515396075530)()
	defer callstack.Trace(515396075530)()

	var checksToValidate []ConstraintToValidate
	for i := 0; i < 10; i++ {
		checksToValidate = append(checksToValidate, ConstraintToValidate{Name: "nil string"})
	}
	sc.validateChecks(checksToValidate)
}

func TestCockroach35501(t *testing.T) {
	//defer callstack.Trace(515396075531)()
	defer callstack.Trace(515396075531)()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		//defer callstack.Trace(515396075532)()
		defer callstack.Trace(515396075532)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
func TestCockroach35501_1(t *testing.T) {
	//d/efer callstack.Trace(515396075533)()
	defer callstack.Trace(515396075531)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		//defer callstack.Trace(515396075534)()
		defer callstack.Trace(515396075532)()
		defer wg.Done()
		sc := &SchemaChanger{}
		sc.runBackfill()
	}()
	wg.Wait()
}
