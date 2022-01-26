package sskiplist

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

type Orderable interface {
	Less(other interface{}) bool
	Equal(other interface{}) bool
}

type SL struct {
	head        *Element
	tail        *Element
	len         int
	maxLevel    int
	levelLookup []int
	randGen     *rand.Rand
}

type Element struct {
	levelLinks []link
	value      Orderable
}

func (e *Element) String() string {
	ss := make([]string, len(e.levelLinks))
	for i, l := range e.levelLinks {
		ss[i] = l.String()
	}
	return fmt.Sprintf("%p %v %s", e, e.value, strings.Join(ss, "\t"))
}

type link struct {
	next   *Element
	offset int
}

func (l link) String() string {
	return fmt.Sprintf("%010p %2d", l.next, l.offset)
}

func New() *SL {
	return NewWithLevel(10)
}

func NewWithLevel(maxLevel int) *SL {
	return &SL{
		maxLevel:    maxLevel,
		levelLookup: probabilityTable(maxLevel),
		randGen:     rand.New(rand.NewSource(7)), // make this deterministic
	}
}

// current height in level is the height of
// the head's levelLinks
func (sl *SL) height() int {
	return len(sl.head.levelLinks)
}

func (sl *SL) Set(v Orderable) {
	//fmt.Println("Starting insert", v)
	e := sl.newElement(v)
	// first insertion
	if sl.head == nil {
		sl.head = e
		sl.len = 1
		return
	}
	// new head
	// swap the links to the new and then insert the old head value as normal
	if v.Less(sl.head.value) {
		oldHead := sl.head
		e.levelLinks = oldHead.levelLinks
		sl.head = e
		e = sl.newElement(oldHead.value)
		//fmt.Println("New head")
		//printList(sl)
	}

	// if the new element increases the current max level
	// for the list, increase the head to match
	if len(e.levelLinks) > sl.height() {
		newLinks := make([]link, len(e.levelLinks))
		for i, l := range sl.head.levelLinks {
			newLinks[i] = l
		}
		for i := len(sl.head.levelLinks); i < len(newLinks); i++ {
			newLinks[i].offset = sl.len
		}
		sl.head.levelLinks = newLinks
		//fmt.Println("New head levels")
		//printList(sl)
	}

	// search from the head
	runner := sl.head
	prevLinks := make([]*link, sl.height())
	// accumulated offsets at each level
	lskips := make([]int, sl.height())
	// TODO remove me
	for i, _ := range lskips {
		lskips[i] = 0
	}
	for i := 0; i < sl.height(); i++ {
		// start at the highest level
		l := sl.height() - 1 - i
		for runner.levelLinks[l].next != nil && runner.levelLinks[l].next.value.Less(e.value) {
			lskips[l] += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
		prevLinks[l] = &runner.levelLinks[l]
	}
	// accumulate the skips for each
	// acclskips := make([]int, len(lskips))
	// for i, ls := range lskips {
	// 	if i == 0 {
	// 		acclskips[i] = ls
	// 	} else {
	// 		acclskips[i] = acclskips[i-1] + ls
	// 	}
	// }
	// for i, l := range prevLinks {
	// 	fmt.Printf("%v %d %p\n", v, i, l.next)
	// }
	// fmt.Println()
	//fmt.Println("Inserting", e.value)
	//fmt.Printf("lskips: %v\n", lskips)
	//fmt.Printf("acclskips: %v\n", acclskips)
	// prevLinksStr := make([]string, len(prevLinks))
	// for i, l := range prevLinks {
	// 	prevLinksStr[i] = l.String()
	// }
	// fmt.Printf("prevs: %v\n", strings.Join(prevLinksStr, "\t"))

	// this accumulates all the nodes skipped prior to this insertion
	// for all levels so that the previous links for thos levels can be updated
	accLevelSkips := lskips[0]
	for i, _ := range e.levelLinks {
		e.levelLinks[i] = *prevLinks[i]
		if i == 0 {
			e.levelLinks[i].offset = 1
		} else {
			e.levelLinks[i].offset = (prevLinks[i].offset + 1) - (accLevelSkips + 1) // - (acclskips[i-1] + 1)
		}
		prevLinks[i].next = e
		if i > 0 {
			prevLinks[i].offset = accLevelSkips + 1 //acclskips[i-1] + 1
		}
		accLevelSkips += lskips[i]
	}
	// these are links above the current insertion level
	// do +1 to cover the insertion
	for i := len(e.levelLinks); i < len(prevLinks); i++ {
		prevLinks[i].offset++
	}
	sl.len++
}

func (sl *SL) Get(v Orderable) (int, *Element) {
	if sl.head == nil {
		return 0, nil
	}
	if v.Less(sl.head.value) {
		return 0, nil
	}
	runner := sl.head
	indexCounter := 0
	height := sl.height()
	//var curlink link
	for i := 0; i < height; i++ {
		// start at the highest level
		l := height - 1 - i
		for runner.levelLinks[l].next != nil && runner.levelLinks[l].next.value.Less(v) {
			indexCounter += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
	}
	if runner.value.Equal(v) {
		return indexCounter, runner.levelLinks[0].next
	}
	if runner.levelLinks[0].next.value.Equal(v) {
		return indexCounter + 1, runner.levelLinks[0].next
	}
	return indexCounter + 1, nil
}

func (sl *SL) Remove(v Orderable) (int, *Element) {
	if sl.head == nil {
		return 0, nil
	}
	// remove the head, make the second node the head
	if sl.head.value.Equal(v) {
		oldHead := sl.head
		newHead := sl.head.levelLinks[0].next
		newLevelLinks := make([]link, len(oldHead.levelLinks))
		// copy whatever levelinks exist on the new head
		for i, l := range newHead.levelLinks {
			newLevelLinks[i] = l
		}
		// if the oldHead has more levels, copy and decrement
		for i := len(newHead.levelLinks); i < len(newLevelLinks); i++ {
			newLevelLinks[i] = oldHead.levelLinks[i]
			newLevelLinks[i].offset--
		}
		newHead.levelLinks = newLevelLinks
		sl.head = newHead
		sl.len--
		return 0, oldHead
	}
	// search from the head
	runner := sl.head
	prevLinks := make([]*link, sl.height())
	indexCounter := 0
	for i := 0; i < sl.height(); i++ {
		// start at the highest level
		l := sl.height() - 1 - i
		for runner.levelLinks[l].next != nil && runner.levelLinks[l].next.value.Less(v) {
			indexCounter += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
		prevLinks[l] = &runner.levelLinks[l]
	}
	// didn't find it
	removeMe := runner.levelLinks[0].next
	if removeMe == nil || !removeMe.value.Equal(v) {
		return 0, nil
	}

	// relink all the levels
	for i, pl := range prevLinks {
		newLinkOffset := 1
		if i > 0 {
			removedOffset := 0
			if i < len(removeMe.levelLinks) {
				removedOffset = removeMe.levelLinks[i].offset
			}
			newLinkOffset = pl.offset + removedOffset - 1
		}
		pl.offset = newLinkOffset
		if pl.next == removeMe {
			pl.next = removeMe.levelLinks[i].next
		}
	}
	sl.len--
	return indexCounter, removeMe
}

func (sl *SL) newElement(v Orderable) *Element {
	l := sl.randLevel()
	ll := make([]link, l+1)
	for i, _ := range ll {
		ll[i].offset = 1
	}
	return &Element{
		levelLinks: ll,
		value:      v,
	}
}

func printList(sl *SL) {
	runner := sl.head
	i := 0
	for runner != nil {
		fmt.Printf("%d %v\n", i, runner)
		runner = runner.levelLinks[0].next
		i++
	}
	fmt.Println()
}

const (
	// Suitable for math.Floor(math.Pow(math.E, 18)) == 65659969 elements in list
	DefaultMaxLevel int = 18
	DefaultProbability
)

func (sl *SL) randLevel() (level int) {
	r := sl.randGen.Int()
	for i := 1; i < len(sl.levelLookup); i++ {
		if r > sl.levelLookup[i] {
			return i - 1
		}
	}
	return len(sl.levelLookup) - 1
}

// probabilityTable calculates in advance the probabilities
// along (1/e)^x
func probabilityTable(level int) []int {
	table := make([]int, level)
	var probability float64 = 1 / math.E
	// the first element is always MaxInt as we always fill in the zero level.
	// also the math below will overflow and wrap negative
	table[0] = math.MaxInt
	for i := 1; i < level; i++ {
		prob := math.Pow(probability, float64(i))
		table[i] = int(math.Floor(math.MaxInt * prob))
	}
	return table
}
