package main

import "fmt"
import "github.com/aws/aws-sdk-go/service/ec2"

type Worker struct {
	dispatcherQueue chan chan *ec2.Instance
	workChannel     chan *ec2.Instance
	quit            chan bool
}

func NewWorker(dispatcher chan chan *ec2.Instance) *Worker {
	return &Worker{
		dispatcherQueue: dispatcher,
		workChannel:     make(chan *ec2.Instance),
		quit:            make(chan bool, 1),
	}
}

func (w *Worker) GetQueueLength() int {
	return len(w.workChannel)
}

func (w *Worker) Start(db *Db, idx int) {
	go func() {
		for {
			w.dispatcherQueue <- w.workChannel
			select {
			case i := <-w.workChannel:
				fmt.Printf("Working instance: %s\n", *i.InstanceId)
				dbInstance, err := db.GetInstanceById(*i.InstanceId)
				if err != nil {
					_, err = db.InsertInstance(i)
					if err != nil {
						fmt.Printf("[%d] Failed to insert instance (%s): %s\n", idx, *i.InstanceId, err)
					}
					continue
				}
				if dbInstance != nil {
					if diff(dbInstance, i) != 0 {
						_, err = db.UpdateInstanceData(i)
						if err != nil {
							fmt.Printf("[%d] failed to update instance: %s: %s\n", idx, *i.InstanceId, err)
						}
					} else {
						db.UpdateInstanceTimestamp(*i.InstanceId)
						if err != nil {
							fmt.Printf("[%d] failed to update timestamp: %s: %s\n", idx, *i.InstanceId, err)
						}
					}
				} else {
					_, err = db.InsertInstance(i)
					if err != nil {
						fmt.Printf("[%d] Failed to insert instance %s: %s\n", idx, *i.InstanceId, err)
					}
				}
				fmt.Printf("Completed instance: %s\n", *i.InstanceId)
			case <-w.quit:
				fmt.Printf("Worker %d exiting\n", idx)
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	w.quit <- true
}
