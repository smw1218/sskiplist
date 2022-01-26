package sskiplist

import (
	"fmt"
	"testing"
)

type OrderableInt int

func (io OrderableInt) Equal(other interface{}) bool {
	return io == other.(OrderableInt)
}

func (io OrderableInt) Less(other interface{}) bool {
	return io < other.(OrderableInt)
}

func BenchmarkGet(b *testing.B) {
	size := 1000000
	sl := NewWithLevel(10)
	for i := 0; i < size; i++ {
		sl.Set(OrderableInt(i))
	}
	// f, err := os.Create("get_profile.out")
	// if err != nil {
	// 	b.Fatalf("Failed prof create: %v", err)
	// }
	b.ResetTimer()
	//pprof.StartCPUProfile(f)
	for i := 0; i < b.N; i++ {
		sl.Get(OrderableInt(i % size))
	}
	//pprof.StopCPUProfile()
}

func TestFull(t *testing.T) {
	stuff := []OrderableInt{7, 5, 2, 9, 1, 3, 4, 6}
	//stuff := []OrderableInt{7, 5, 2, 9, 1}
	sl := NewWithLevel(5)
	for _, v := range stuff {
		sl.Set(v)

	}
	printList(sl)
	idx, e := sl.Get(OrderableInt(7))
	fmt.Printf("Get %d %v\n", idx, e)

	idx, e = sl.Get(OrderableInt(1))
	fmt.Printf("Get %d %v\n", idx, e)

	idx, e = sl.Get(OrderableInt(9))
	fmt.Printf("Get %d %v\n", idx, e)

	idx, e = sl.Remove(OrderableInt(5))
	fmt.Printf("Rem %d %v\n", idx, e)
	printList(sl)

	idx, e = sl.Remove(OrderableInt(1))
	fmt.Printf("Rem %d %v\n", idx, e)
	printList(sl)
}

func TestProbTable(t *testing.T) {
	t.Skip()
	sl := NewWithLevel(10)
	table := sl.levelLookup
	for i, p := range table {
		fmt.Printf("%d\t%d\n", i, p)
	}
	fmt.Println("")
	counts := make([]int, len(table))
	for i := 0; i < 100000; i++ {
		l := sl.randLevel()
		counts[l]++
		//fmt.Printf("%d %d\n", i, l)
	}
	for i, c := range counts {
		fmt.Printf("%d\t%d\n", i, c)
	}
	t.Error("FAIL")
}
