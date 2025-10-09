package chunked

type Chunk struct {
	Index int
	Data  []byte
	Error error
}

type ChunkHeap []*Chunk

func (h ChunkHeap) Len() int           { return len(h) }
func (h ChunkHeap) Less(i, j int) bool { return h[i].Index < h[j].Index }
func (h ChunkHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ChunkHeap) Push(x interface{}) {
	*h = append(*h, x.(*Chunk))
}

func (h *ChunkHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
