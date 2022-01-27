package sskiplist

import (
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
)

type OrderableInt int

func (io OrderableInt) Equal(other interface{}) bool {
	return io == other.(OrderableInt)
}

func (io OrderableInt) Less(other interface{}) bool {
	return io < other.(OrderableInt)
}

const testingSize = 500000

func TestSetLevelDistribution(t *testing.T) {
	t.Skip()
	size := testingSize
	sl := NewWithLevel(18)
	for i := 0; i < size; i++ {
		sl.Set(OrderableInt(i))
	}
	counts := make([]int, sl.height())
	runner := sl.head
	for runner != nil {
		counts[len(runner.levelLinks)-1]++
		runner = runner.levelLinks[0].next
	}
	fmt.Println("Set dist")
	for i, c := range counts {
		fmt.Printf("x %d\t%d\n", i, c)
	}
	fmt.Println()
}

func BenchmarkGet(b *testing.B) {
	//b.Skip()
	size := testingSize
	sl := NewWithLevel(18)
	for i := 0; i < size; i++ {
		sl.Set(OrderableInt(i))
	}
	f, err := os.Create("get_profile.out")
	if err != nil {
		b.Fatalf("Failed prof create: %v", err)
	}
	b.ResetTimer()
	pprof.StartCPUProfile(f)
	for i := 0; i < b.N; i++ {
		sl.Get(OrderableInt(i % size))
	}
	pprof.StopCPUProfile()
}

func BenchmarkSet(b *testing.B) {
	//b.Skip()
	f, err := os.Create("set_profile.out")
	if err != nil {
		b.Fatalf("Failed prof create: %v", err)
	}

	sl := NewWithLevel(18)

	b.ResetTimer()
	pprof.StartCPUProfile(f)
	for i := 0; i < b.N; i++ {
		sl.Set(OrderableInt(i))
	}
	pprof.StopCPUProfile()
}
func TestFull(t *testing.T) {
	stuff := []OrderableInt{7, 5, 2, 9, 1, 3, 4, 6}
	//stuff := []OrderableInt{7, 5, 2, 9, 1}
	lenstuff := len(stuff)
	sl := NewWithLevel(10)
	for _, v := range stuff {
		sl.Set(v)

		err := sl.checkOffsets()
		if err != nil {
			printList(sl)
			t.Fatalf("Corrupted list from Set %v: %v", v, err)
		}
	}
	printList(sl)
	if sl.Size() != lenstuff {
		t.Errorf("Wrong size %d, expected %d", sl.Size(), lenstuff)
	}

	idx, e := sl.Get(OrderableInt(7))
	t.Logf("Get val 7 %d %v\n", idx, e)

	idx, e = sl.Get(OrderableInt(1))
	t.Logf("Get val 1 %d %v\n", idx, e)

	idx, e = sl.Get(OrderableInt(9))
	t.Logf("Get %d %v\n", idx, e)

	idx, e = sl.Remove(OrderableInt(5))
	t.Logf("Rem %d %v\n", idx, e)
	err := sl.checkOffsets()
	if err != nil {
		printList(sl)
		t.Fatalf("Corrupted list from Rem 5: %v", err)
	}
	if sl.Size() != lenstuff-1 {
		t.Errorf("Failed remove size %v, expected %v", sl.Size(), lenstuff-1)
	}

	idx, e = sl.Remove(OrderableInt(1))
	t.Logf("Rem %d %v\n", idx, e)
	err = sl.checkOffsets()
	if err != nil {
		printList(sl)
		t.Fatalf("Corrupted list from Rem 1: %v", err)
	}
	if sl.Size() != lenstuff-2 {
		t.Errorf("Failed remove size %v, expected %v", sl.Size(), lenstuff-2)
	}
}

//func BenchmarkTestProbTable(t *testing.B) {
func TestProbTable(t *testing.T) {
	t.Skip()
	sl := New()
	table := sl.levelLookup
	// for i, p := range table {
	// 	fmt.Printf("%d\t%d\n", i, p)
	// }
	// fmt.Println("")
	counts := make([]int, len(table))
	//for i := 0; i < t.N; i++ {
	for i := 0; i < 500000; i++ {
		l := sl.randLevel()
		counts[l]++
		//fmt.Printf("%d %d\n", i, l)
	}
	for i, c := range counts {
		fmt.Printf("x %d\t%d\n", i, c)
	}
	t.Error("FAIL")
}
