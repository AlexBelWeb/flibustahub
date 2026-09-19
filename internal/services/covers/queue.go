package covers

import (
	"container/heap"
	"context"
	"sync"
)

// Need is the set of archive products a job should extract.
type Need int

const (
	NeedCover Need = 1 << iota
	NeedAnnotation
)

func (n Need) has(flag Need) bool { return n&flag != 0 }

// Priority is the user-attention order. Lower values run first.
type Priority int

const (
	PrioOpen Priority = iota
	PrioVisible
	PrioPrefetch
	PrioWarmup
)

type waiter struct {
	need Need
	ctx  context.Context
	ch   chan error
}

type job struct {
	workID      int64
	need        Need
	prio        Priority
	seq         uint64
	index       int
	started     bool
	waiters     []waiter
	pending     Need
	pendingPrio Priority
	pendingWait []waiter
}

type jobHeap []*job

func (h jobHeap) Len() int { return len(h) }

func (h jobHeap) Less(i, j int) bool {
	a, b := h[i], h[j]
	if a.prio != b.prio {
		return a.prio < b.prio
	}
	if a.prio == PrioWarmup {
		return a.seq < b.seq
	}
	return a.seq > b.seq
}

func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *jobHeap) Push(x any) {
	j := x.(*job)
	j.index = len(*h)
	*h = append(*h, j)
}

func (h *jobHeap) Pop() any {
	old := *h
	n := len(old)
	j := old[n-1]
	old[n-1] = nil
	j.index = -1
	*h = old[:n-1]
	return j
}

type queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	h      jobHeap
	inf    map[int64]*job
	seq    uint64
	closed bool
}

func newQueue() *queue {
	q := &queue{inf: make(map[int64]*job)}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.h)
	return q
}

func (q *queue) Enqueue(ctx context.Context, workID int64, need Need, prio Priority) <-chan error {
	ch := make(chan error, 1)
	if need == 0 || workID <= 0 {
		ch <- nil
		return ch
	}
	w := waiter{need: need, ctx: ctx, ch: ch}

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		ch <- context.Canceled
		return ch
	}
	if inf, ok := q.inf[workID]; ok {
		missing := need &^ inf.need
		if missing == 0 {
			inf.waiters = append(inf.waiters, w)
			q.raise(inf, prio)
			return ch
		}
		if !inf.started {
			inf.need |= need
			inf.waiters = append(inf.waiters, w)
			q.raise(inf, prio)
			return ch
		}
		inf.pending |= missing
		if len(inf.pendingWait) == 0 || prio < inf.pendingPrio {
			inf.pendingPrio = prio
		}
		inf.pendingWait = append(inf.pendingWait, w)
		return ch
	}
	q.seq++
	j := &job{workID: workID, need: need, prio: prio, seq: q.seq, waiters: []waiter{w}}
	q.inf[workID] = j
	heap.Push(&q.h, j)
	q.cond.Signal()
	return ch
}

func (q *queue) raise(j *job, prio Priority) {
	if prio < j.prio {
		j.prio = prio
		if j.index >= 0 {
			heap.Fix(&q.h, j.index)
		}
	}
}

func (q *queue) Take() *job {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		if q.closed {
			return nil
		}
		if q.h.Len() > 0 {
			j := heap.Pop(&q.h).(*job)
			if abandoned(j) {
				if cur := q.inf[j.workID]; cur == j {
					delete(q.inf, j.workID)
				}
				for _, w := range j.waiters {
					notify(w, context.Canceled)
				}
				for _, w := range j.pendingWait {
					notify(w, context.Canceled)
				}
				continue
			}
			j.started = true
			j.index = -1
			return j
		}
		q.cond.Wait()
	}
}

func (q *queue) Done(j *job, err error) {
	if j == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if cur := q.inf[j.workID]; cur == j {
		delete(q.inf, j.workID)
	}
	for _, w := range j.waiters {
		notify(w, err)
	}
	if err != nil || j.pending == 0 {
		for _, w := range j.pendingWait {
			notify(w, err)
		}
		return
	}
	q.seq++
	rest := &job{
		workID:  j.workID,
		need:    j.pending,
		prio:    j.pendingPrio,
		seq:     q.seq,
		waiters: j.pendingWait,
	}
	q.inf[j.workID] = rest
	heap.Push(&q.h, rest)
	q.cond.Signal()
}

func (q *queue) Cancel(workID int64, ch <-chan error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	j, ok := q.inf[workID]
	if !ok {
		return
	}
	j.waiters = dropWaiter(j.waiters, ch)
	j.pendingWait = dropWaiter(j.pendingWait, ch)
	if j.started {
		return
	}
	if len(j.waiters) > 0 || len(j.pendingWait) > 0 {
		return
	}
	delete(q.inf, workID)
	if j.index >= 0 && j.index < q.h.Len() {
		heap.Remove(&q.h, j.index)
	}
}

func (q *queue) DropUnstartedWarmup() {
	q.mu.Lock()
	defer q.mu.Unlock()
	keep := make(jobHeap, 0, q.h.Len())
	for _, j := range q.h {
		if j == nil {
			continue
		}
		if j.prio == PrioWarmup && !j.started {
			delete(q.inf, j.workID)
			for _, w := range j.waiters {
				notify(w, context.Canceled)
			}
			for _, w := range j.pendingWait {
				notify(w, context.Canceled)
			}
			continue
		}
		keep = append(keep, j)
	}
	q.h = keep
	heap.Init(&q.h)
}

func (q *queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	for _, j := range q.inf {
		for _, w := range j.waiters {
			notify(w, context.Canceled)
		}
		for _, w := range j.pendingWait {
			notify(w, context.Canceled)
		}
	}
	q.inf = make(map[int64]*job)
	q.h = nil
	q.cond.Broadcast()
}

func dropWaiter(list []waiter, ch <-chan error) []waiter {
	out := list[:0]
	for _, w := range list {
		if w.ch != ch {
			out = append(out, w)
		}
	}
	return out
}

func notify(w waiter, err error) {
	select {
	case w.ch <- err:
	default:
	}
}

func abandoned(j *job) bool {
	j.waiters = liveWaiters(j.waiters)
	j.pendingWait = liveWaiters(j.pendingWait)
	return len(j.waiters) == 0 && len(j.pendingWait) == 0
}

func liveWaiters(list []waiter) []waiter {
	out := list[:0]
	for _, w := range list {
		if w.ctx == nil || w.ctx.Err() == nil {
			out = append(out, w)
		}
	}
	return out
}
