package main

import "fmt"
import "os"
import "github.com/aws/aws-sdk-go/service/ec2"
import "runtime"
import "strconv"
import "time"

func main() {

	// Some initialization to determine the number of worker threads
	MaxWorkersEnv := os.Getenv("CLOUDIE_MAX_WORKERS")
	MaxWorkers, err := strconv.Atoi(MaxWorkersEnv)
	if err != nil {
		MaxWorkers = 4
	}
	WorkQueue := make(chan *ec2.Instance, MaxWorkers)

	ec2 := Ec2Init("us-east-1")

	d := NewDispatcher(MaxWorkers)
	d.Run(WorkQueue)

	err = ec2.DescribeInstances(WorkQueue)
	if err != nil {
		fmt.Printf("Failed getting the instances: %s\n", err)
	}

	emptyCount := 0
	for {
		if len(WorkQueue) == 0 && d.GetWork() == 0 {
			emptyCount++
			if emptyCount > 3 {
				break
			}
			time.Sleep(1 * time.Second)
		} else {
			emptyCount = 0
			runtime.Gosched()
		}
	}

	// Give the workers another second to complete any pending work
	time.Sleep(1 * time.Second)
	d.Stop()
}
