package covers

import (
	"context"
	"testing"
	"time"
)

func TestEnqueueExtendsUnstartedNeed(t *testing.T) {
	q := newQueue()
	t.Cleanup(q.Close)
	ctx := context.Background()
	ch1 := q.Enqueue(ctx, 7, NeedCover, PrioVisible)
	ch2 := q.Enqueue(ctx, 7, NeedCover|NeedAnnotation, PrioOpen)
	j := q.Take()
	if j == nil {
		t.Fatal("no job")
	}
	if j.need != NeedCover|NeedAnnotation {
		t.Fatalf("need=%b", j.need)
	}
	if j.prio != PrioOpen {
		t.Fatalf("prio=%d", j.prio)
	}
	if q.h.Len() != 0 {
		t.Fatal("second job queued despite extend")
	}
	q.Done(j, nil)
	if err := <-ch1; err != nil {
		t.Fatal(err)
	}
	if err := <-ch2; err != nil {
		t.Fatal(err)
	}
}

func TestStartedJobLeavesRemainder(t *testing.T) {
	q := newQueue()
	t.Cleanup(q.Close)
	ctx := context.Background()
	chCover := q.Enqueue(ctx, 3, NeedCover, PrioVisible)
	first := q.Take()
	if first == nil || first.need != NeedCover || !first.started {
		t.Fatalf("first=%+v", first)
	}
	chBoth := q.Enqueue(ctx, 3, NeedCover|NeedAnnotation, PrioOpen)
	if q.h.Len() != 0 {
		t.Fatal("remainder must wait until the started job finishes")
	}
	got := make(chan *job, 1)
	go func() { got <- q.Take() }()
	q.Done(first, nil)
	if err := <-chCover; err != nil {
		t.Fatal(err)
	}
	second := <-got
	if second == nil {
		t.Fatal("expected remainder job")
	}
	if second.need != NeedAnnotation {
		t.Fatalf("remainder need=%b", second.need)
	}
	q.Done(second, nil)
	if err := <-chBoth; err != nil {
		t.Fatal(err)
	}
}

func TestCanceledUnstartedIsDropped(t *testing.T) {
	q := newQueue()
	t.Cleanup(q.Close)
	alive := context.Background()
	dead, cancel := context.WithCancel(context.Background())
	q.Enqueue(alive, 9, NeedCover, PrioVisible)
	q.Enqueue(dead, 10, NeedCover, PrioVisible)
	cancel()
	j := q.Take()
	if j == nil || j.workID != 9 {
		t.Fatalf("got %+v, want uncancelled work 9", j)
	}
	q.Done(j, nil)
}

func TestPriorityOpenBeforeVisible(t *testing.T) {
	q := newQueue()
	t.Cleanup(q.Close)
	ctx := context.Background()
	q.Enqueue(ctx, 1, NeedCover, PrioVisible)
	q.Enqueue(ctx, 2, NeedCover, PrioOpen)
	j := q.Take()
	if j.workID != 2 {
		t.Fatalf("work=%d", j.workID)
	}
	q.Done(j, nil)
	j = q.Take()
	if j.workID != 1 {
		t.Fatalf("work=%d", j.workID)
	}
	q.Done(j, nil)
}

func TestWarmupFIFOInteractiveLIFO(t *testing.T) {
	q := newQueue()
	t.Cleanup(q.Close)
	ctx := context.Background()
	q.Enqueue(ctx, 1, NeedCover, PrioWarmup)
	time.Sleep(time.Millisecond)
	q.Enqueue(ctx, 2, NeedCover, PrioWarmup)
	q.Enqueue(ctx, 3, NeedCover, PrioVisible)
	q.Enqueue(ctx, 4, NeedCover, PrioVisible)
	j := q.Take()
	if j.workID != 4 {
		t.Fatalf("newest visible first, got %d", j.workID)
	}
	q.Done(j, nil)
	j = q.Take()
	if j.workID != 3 {
		t.Fatalf("older visible next, got %d", j.workID)
	}
	q.Done(j, nil)
	j = q.Take()
	if j.workID != 1 {
		t.Fatalf("warmup FIFO first, got %d", j.workID)
	}
	q.Done(j, nil)
	j = q.Take()
	if j.workID != 2 {
		t.Fatalf("warmup FIFO second, got %d", j.workID)
	}
	q.Done(j, nil)
}
