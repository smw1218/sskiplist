# Scott's Skiplist

TBH, there are probably better/faster implementations out there. 
See https://github.com/MauriceGit/skiplist-survey

I wrote this one because I needed an [indexable skiplist](https://en.wikipedia.org/wiki/Skip_list#Indexable_skiplist)
and [this one](https://github.com/glenn-brown/skiplist) has a weird license (looks a bit like the Redis license?). I also wanted a list that could have multiple of the same score (which many don't support).

Items in the skiplist need to implement the `Orderable` interface:

    type Orderable interface {
        Less(other interface{}) bool
        Equal(other interface{}) bool
    }

This means there's no separation of the score and the stored object. Just implement the interface
on anything and you're good to go.

    type OrderableInt int

    func (io OrderableInt) Equal(other interface{}) bool {
        return io == other.(OrderableInt)
    }

    func (io OrderableInt) Less(other interface{}) bool {
        return io < other.(OrderableInt)
    }

    func main() {
        sl := sskiplist.New()
        sl.Set(OrderableInt(17))
        sl.Set(OrderableInt(7))
        sl.Set(OrderableInt(77))
        idx, e := sl.Get(OrderableInt(17))
        fmt.Printf("sl[%d] = %v\n", idx, e.Value)
    }