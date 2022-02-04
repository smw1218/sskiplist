package sskiplist

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// Orderable defines and interface for a value that
// can be inserted into the SL.
type Orderable interface {
	Less(other interface{}) bool
	Equal(other interface{}) bool
}

// SL is an indexable skip list
// it is not goroutine safe so needs
// external synchronization.
type SL struct {
	head        *Element
	len         int
	maxLevel    int
	levelLookup []int
	randGen     *rand.Rand
	// pre-allocations; this is very explicitly not goroutine safe
	prev   []linkHolder
	lskips []int
}

// Element is a wrapper around the Orderable.
// Calling Prev/Next on an Element after any modification to
// the skip list with Set or Remove has nondeterministic behavior.
// i.e, if you are locking for goroutines, you should do all the Prev/Next calls
// in the same read-lock as the Get
type Element struct {
	levelLinks links
	Value      Orderable
}

func (e *Element) String() string {
	return fmt.Sprintf("%p %v %s", e, e.Value, e.levelLinks)
}

func (e *Element) Prev() *Element {
	return e.levelLinks[0].previous
}

func (e *Element) Next() *Element {
	return e.levelLinks[0].next
}

type link struct {
	next     *Element
	previous *Element
	offset   int
}

func (l link) String() string {
	return fmt.Sprintf("%010p %2d", l.previous, l.offset)
}

type links []link

func (ls links) String() string {
	ss := make([]string, len(ls))
	for i, l := range ls {
		ss[i] = l.String()
	}
	return strings.Join(ss, "\t")
}

// used for storing intermediate prev links
// while traversing the skiplist
type linkHolder struct {
	prevLink *link
	element  *Element
}

// New creates a SL prepared with the DefaultMaxLevel
func New() *SL {
	return NewWithLevel(DefaultMaxLevel)
}

// NewWithLevel creates a SL prepared with
// the passed maxLevel
func NewWithLevel(maxLevel int) *SL {
	return &SL{
		maxLevel:    maxLevel,
		levelLookup: probabilityTable(maxLevel),
		randGen:     rand.New(rand.NewSource(7)), // make this deterministic
		prev:        make([]linkHolder, maxLevel),
		lskips:      make([]int, maxLevel),
	}
}

// current height in level is the height of
// the head's levelLinks
func (sl *SL) height() int {
	if sl.head == nil {
		return 0
	}
	return len(sl.head.levelLinks)
}

func (sl *SL) Size() int {
	return sl.len
}

// Set adds v to the SL and returns the index and Element for the
// insertion
func (sl *SL) Set(v Orderable) (int, *Element) {
	//fmt.Println("Starting insert", v)
	e := sl.newElement(v)
	// first insertion
	if sl.head == nil {
		sl.head = e
		for i := range e.levelLinks {
			e.levelLinks[i].offset = 1
		}
		sl.len = 1
		return 0, e
	}
	// new head
	// swap the links to the new and then insert the old head value as normal
	newHead := false
	if v.Less(sl.head.Value) {
		oldHead := sl.head
		e.levelLinks = oldHead.levelLinks
		sl.head = e
		e = sl.newElement(oldHead.Value)
		//fmt.Println("New head")
		//printList(sl)
		newHead = true
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

	indexCounter, _, prevLinks, lskips := sl.prevWithLinks(e.Value)

	// prevStrings := make([]string, len(prevLinks))
	// for i, p := range prevLinks {
	// 	prevStrings[i] = p.String()
	// }
	// fmt.Printf("prevs: %v\n", strings.Join(prevStrings, " | "))
	// fmt.Printf("lskips: %v\n", lskips)

	// this accumulates all the nodes skipped prior to this insertion
	// for all levels so that the previous links for those levels can be updated
	accLevelSkips := lskips[0]

	// relink and update offsets
	for i := range prevLinks {
		if i < len(e.levelLinks) {
			// update the links
			e.levelLinks[i] = *prevLinks[i].prevLink
			e.levelLinks[i].previous = prevLinks[i].element
			if e.levelLinks[i].next != nil {
				e.levelLinks[i].next.levelLinks[i].previous = e
			}
			prevLinks[i].prevLink.next = e

			// update the link offsets
			if i > 0 {
				e.levelLinks[i].offset = (prevLinks[i].prevLink.offset + 1) - (accLevelSkips + 1)
				prevLinks[i].prevLink.offset = accLevelSkips + 1
				accLevelSkips += lskips[i]
			} else {
				e.levelLinks[i].offset = 1
			}
		} else {
			// these are links above the current insertion level
			// do +1 to cover the insertion
			prevLinks[i].prevLink.offset++
		}
	}
	sl.len++
	if newHead {
		return 0, sl.head
	}
	return indexCounter + 1, e
}

func (sl *SL) prevWithLinks(v Orderable) (indexCounter int, e *Element, prev []linkHolder, lskips []int) {
	// search from the head
	runner := sl.head
	//prev = make([]*link, sl.height())
	// accumulated offsets at each level
	//lskips = make([]int, sl.height())
	height := sl.height()
	sl.resetPrevs(height)
	for i := 0; i < height; i++ {
		// start at the highest level
		l := height - 1 - i
		for runner.levelLinks[l].next != nil && runner.levelLinks[l].next.Value.Less(v) {
			sl.lskips[l] += runner.levelLinks[l].offset
			indexCounter += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
		sl.prev[l].prevLink = &runner.levelLinks[l]
		sl.prev[l].element = runner
	}

	return indexCounter, runner, sl.prev[:height], sl.lskips[:height]
}

func (sl *SL) resetPrevs(height int) {
	for i := 0; i < height; i++ {
		sl.prev[i].prevLink = nil
		sl.prev[i].element = nil
		sl.lskips[i] = 0
	}
}

// for testing
func (sl *SL) checkOffsets() error {
	runner := sl.head
	offsetSums := make([]int, sl.height())
	for runner != nil {
		for i, l := range runner.levelLinks {
			if l.offset < 1 {
				return fmt.Errorf("invalid link offset at: %v", runner)
			}
			offsetSums[i] += l.offset
		}
		runner = runner.levelLinks[0].next
	}
	for i, levelSum := range offsetSums {
		if levelSum != sl.len {
			return fmt.Errorf("level %d incorrect sum %v, expected %v", i, levelSum, sl.len)
		}
	}
	return nil
}

// GetAt gets the item at the specific index. This returns nil if the index
// is out of bounds.
func (sl *SL) GetAt(index int) *Element {
	if index == 0 {
		return sl.head
	}
	if sl.head == nil {
		return nil
	}
	if index >= sl.len {
		return nil
	}
	runner := sl.head
	indexCounter := 0
	for l := sl.height() - 1; l >= 0; l-- {
		for runner != nil && indexCounter+runner.levelLinks[l].offset <= index {
			indexCounter += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
	}
	return runner
}

// Get attempt find the item where v.Equal is true
// Element returned and the index as well. If no Equal item is found,
// this returns 0, nil
func (sl *SL) Get(v Orderable) (int, *Element) {
	if sl.head == nil {
		return 0, nil
	}
	if v.Less(sl.head.Value) {
		return 0, nil
	}
	indexCounter, runner := sl.prevElement(v)
	if runner.Value.Equal(v) {
		return indexCounter, runner.levelLinks[0].next
	}
	if runner.levelLinks[0].next != nil && runner.levelLinks[0].next.Value.Equal(v) {
		return indexCounter + 1, runner.levelLinks[0].next
	}
	return indexCounter + 1, nil
}

func (sl *SL) prevElement(v Orderable) (int, *Element) {
	runner := sl.head
	indexCounter := 0
	// start at the highest level
	for l := sl.height() - 1; l >= 0; l-- {
		for runner.levelLinks[l].next != nil && runner.levelLinks[l].next.Value.Less(v) {
			indexCounter += runner.levelLinks[l].offset
			runner = runner.levelLinks[l].next
		}
	}
	return indexCounter, runner
}

// Remove will remove a single item where v.Equal is true.
// If a removal is successful, the returned Element will be the
// Element returned and the index as well. If no removal occurred,
// this returns 0, nil
func (sl *SL) Remove(v Orderable) (int, *Element) {
	if sl.head == nil {
		return 0, nil
	}
	// remove the head, make the second node the head
	if sl.head.Value.Equal(v) {
		oldHead := sl.head
		newHead := sl.head.levelLinks[0].next
		if newHead != nil {
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
		}
		sl.head = newHead
		sl.len--
		return 0, oldHead
	}
	// search from the head
	indexCounter, runner, prevLinks, _ := sl.prevWithLinks(v)

	// didn't find it
	removeMe := runner.levelLinks[0].next
	if removeMe == nil || !removeMe.Value.Equal(v) {
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
			newLinkOffset = pl.prevLink.offset + removedOffset - 1
		}
		pl.prevLink.offset = newLinkOffset
		if pl.prevLink.next == removeMe {
			pl.prevLink.next = removeMe.levelLinks[i].next
			if pl.prevLink.next != nil {
				pl.prevLink.next.levelLinks[i].previous = removeMe.levelLinks[i].previous
			}
		}

	}
	sl.len--
	return indexCounter, removeMe
}

func (sl *SL) newElement(v Orderable) *Element {
	l := sl.randLevel()
	return &Element{
		levelLinks: make([]link, l+1),
		Value:      v,
	}
}

// PrintList is mostly for debugging and prints
// the entire SL with all the level links (you've been warned).
// Each element is preintg per line so the levels grow to the right
// rather than the wikipedia horizonatal elements.
func PrintList(sl *SL) {
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
	DefaultMaxLevel int = 18
	// DefaultProbability inverse so 2 => 1/2 as likely for each subsequent level
	DefaultProbability = 2
)

// returns the level index (zero-based)
// it restricts growing the height of the list by one level at a time;
// no good reason other than I saw another SL do this
func (sl *SL) randLevel() int {
	r := sl.randGen.Int()
	height := sl.height()
	for i := 1; i < len(sl.levelLookup); i++ {
		if r > sl.levelLookup[i] || i > height {
			return i - 1
		}
	}
	return len(sl.levelLookup) - 1
}

// probabilityTable calculates in advance the probabilities
func probabilityTable(level int) []int {
	table := make([]int, level)
	// the first element is always MaxInt as we always fill in the zero level.
	// also the math below will overflow and wrap negative
	table[0] = math.MaxInt
	for i := 1; i < level; i++ {
		prob := math.Pow(2, float64(-i))
		table[i] = int(math.Floor(math.MaxInt * prob))
	}
	return table
}
