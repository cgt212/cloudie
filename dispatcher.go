package main

import "fmt"
import "github.com/aws/aws-sdk-go/service/ec2"

type Dispatcher struct {
	workerQueue chan chan *ec2.Instance
	workPool    []*Worker
	db          *Db
	quit        chan bool
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	dbConn, err := DbConnect(maxWorkers)
	if err != nil {
		fmt.Printf("Unable to connect to database: %s\n", err)
		return nil
	}
	dispatcher := &Dispatcher{
		workerQueue: make(chan chan *ec2.Instance, maxWorkers),
		workPool:    make([]*Worker, maxWorkers),
		db:          dbConn,
		quit:        make(chan bool),
	}

	return dispatcher
}

func (d *Dispatcher) Run(workQueue chan *ec2.Instance) {
	for iter := 0; iter < len(d.workPool); iter++ {
		d.workPool[iter] = NewWorker(d.workerQueue)
		d.workPool[iter].Start(d.db, iter)
	}

	go d.dispatch(workQueue)
}

func (d *Dispatcher) dispatch(workQueue chan *ec2.Instance) {
	defer d.db.Close()
	for {
		select {
		case instance := <-workQueue:
			go func(i *ec2.Instance) {
				workerChannel := <-d.workerQueue
				workerChannel <- instance
			}(instance)
		case <-d.quit:
			fmt.Printf("dispatcher exiting.\n")
			return
		}
	}
}

func (d *Dispatcher) GetWork() int {
	qlen := 0
	for _, w := range d.workPool {
		qlen += w.GetQueueLength()
	}
	return qlen
}

func (d *Dispatcher) Stop() {
	fmt.Printf("Stopping dispatcher...")
	d.quit <- true
	for _, w := range d.workPool {
		w.Stop()
	}
}
