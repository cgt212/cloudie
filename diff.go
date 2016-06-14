package main

import "github.com/aws/aws-sdk-go/aws/awsutil"
import "github.com/aws/aws-sdk-go/service/ec2"

const (
	Ebs int = 1 << iota
	Network
	Sg
	State
	Tags
)

func old_diff(was, is *ec2.Instance) bool {
	return awsutil.DeepEqual(was, is)
}

// Diff checks the pieces of the instance that we care about can be distilled down to a few significant parts that
// could require some action to be taken.  Instead of doing a reflect.DeepEqual across the entire map
// we are only going to look for the changes that we think are important.  Those are:
// - BlockDeviceMappings
// - NetworkInterfaces
// - State
//   - Code
// - Security Groups
// - Tags
// For the first 2, we are going to take a shortcut, and try to guage by the length.  The tags might be a little more
// complicated.
func diff(was, is *ec2.Instance) int {
	var ret int = 0
	if is.BlockDeviceMappings != nil && was.BlockDeviceMappings != nil {
		if len(is.BlockDeviceMappings) != len(was.BlockDeviceMappings) {
			ret |= Ebs
		}
	} else {
		//Here we know that both NetworkInterfaces are not defined, so are they both undefined, or different?
		if is.BlockDeviceMappings != nil || was.BlockDeviceMappings != nil {
			ret |= Ebs
		}
	}

	if len(is.NetworkInterfaces) != len(was.NetworkInterfaces) {
		ret |= Network
	}

	if !awsutil.DeepEqual(is.SecurityGroups, was.SecurityGroups) {
		ret |= Sg
	}

	if *is.State.Code != *was.State.Code {
		ret |= State
	}

	if len(is.Tags) != len(was.Tags) {
		ret |= Tags
	} else {
		isTags := map[string]string{}
		wasTags := map[string]string{}
		for i, _ := range is.Tags {
			isTags[*is.Tags[i].Key] = *is.Tags[i].Value
			wasTags[*was.Tags[i].Key] = *was.Tags[i].Value
		}
		for key, value := range isTags {
			v, ok := wasTags[key]
			if !(ok && v == value) {
				ret |= Tags
			}
		}
	}

	return ret
}
