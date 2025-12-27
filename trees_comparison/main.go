package main

import (
	"context"
	"fmt"

	"golang.org/x/tour/tree"
)

func Walk(ctx context.Context, t *tree.Tree, ch chan<- int) {
	if t == nil {
		return
	}

	Walk(ctx, t.Left, ch)
	select {
	case <-ctx.Done():
		return
	case ch <- t.Value:
	}

	Walk(ctx, t.Right, ch)
}

func Same(t1, t2 *tree.Tree) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1 := startWalk(ctx, t1)
	ch2 := startWalk(ctx, t2)

	for {
		v1, ok1 := <-ch1
		v2, ok2 := <-ch2

		if !ok1 && !ok2 {
			return true
		}
		if ok1 != ok2 || v1 != v2 {
			return false
		}
	}
}

func startWalk(ctx context.Context, t *tree.Tree) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		Walk(ctx, t, ch)
	}()
	return ch
}

func main() {
	fmt.Println(Same(tree.New(1), tree.New(1))) // true
	fmt.Println(Same(tree.New(1), tree.New(2))) // false
}
