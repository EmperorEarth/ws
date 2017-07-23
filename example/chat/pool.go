package main

import (
	"fmt"
	"time"
)

var ErrScheduleTimeout = fmt.Errorf("schedule error: timed out")

type Pool struct {
	sem  chan struct{}
	work chan func()
}

func NewPool(start, workers, queue int) *Pool {
	if start <= 0 && queue > 0 {
		panic("dead queue configuration detected")
	}
	if start > workers {
		panic("start > workers")
	}
	p := &Pool{
		sem:  make(chan struct{}, workers),
		work: make(chan func(), queue),
	}
	for i := 0; i < start; i++ {
		p.sem <- struct{}{}
		go p.worker(func() {})
	}

	return p
}

func (p *Pool) Schedule(task func()) {
	p.schedule(task, nil)
}

func (p *Pool) ScheduleTimeout(timeout time.Duration, task func()) error {
	return p.schedule(task, time.After(timeout))
}

func (p *Pool) schedule(task func(), timeout <-chan time.Time) error {
	select {
	case <-timeout:
		return ErrScheduleTimeout
	case p.work <- task:
		return nil
	case p.sem <- struct{}{}:
		go p.worker(task)
		return nil
	}
}

func (p *Pool) worker(task func()) {
	defer func() { <-p.sem }()

	task()

	for task := range p.work {
		task()
	}
}
