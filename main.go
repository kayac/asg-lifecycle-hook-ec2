package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	_ "time/tzdata"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

// LifecycleTransition is transition value.
const LifecycleTransition = "autoscaling:EC2_INSTANCE_TERMINATING"

func main() {
	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		runAsLambda()
	} else {
		runAsCLI()
	}
}

func runAsLambda() {
	lambda.Start(func(ctx context.Context, event *events.AutoScalingEvent) error {
		if err := handler(ctx, event); err != nil {
			log.Println("[error]", err)
			return err
		}
		return nil
	})
}

func runAsCLI() {
	var asgName, instanceID string
	flag.StringVar(&asgName, "asg-name", "", "AutoScalingGroupName")
	flag.StringVar(&instanceID, "instance-id", "", "EC2InstanceId")
	flag.Parse()
	if asgName == "" || instanceID == "" {
		flag.Usage()
		return
	}

	event := &events.AutoScalingEvent{
		Detail: map[string]interface{}{
			"AutoScalingGroupName": asgName,
			"EC2InstanceId":        instanceID,
			"LifecycleTransition":  LifecycleTransition,
		},
	}
	if err := handler(context.Background(), event); err != nil {
		log.Println("[error]", err)
		os.Exit(1)
	}
}

func handler(ctx context.Context, event *events.AutoScalingEvent) error {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return err
	}
	log.Printf("[info] event: %#v", event)

	asgSvc := autoscaling.New(sess)
	asgName := str(event.Detail["AutoScalingGroupName"])
	instanceID := str(event.Detail["EC2InstanceId"])
	transition := str(event.Detail["LifecycleTransition"])
	log.Printf("[info] starting lifecycle hook AutoScalingGroupName:%s EC2InstanceId:%s LifecycleTransition:%s", asgName, instanceID, transition)

	if transition != LifecycleTransition {
		return fmt.Errorf("unexpected transision: %s", transition)
	}
	if err := drainInstance(sess, asgSvc, asgName, instanceID); err != nil {
		return err
	}
	if err := complate(asgSvc, event); err != nil {
		return err
	}

	return nil
}

func drainInstance(sess *session.Session, asgSvc *autoscaling.AutoScaling, asgName string, instanceID string) error {
	// determine the EC2 instance from ELB and ELBv2
	res, err := asgSvc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to DescribeAutoScalingGroups")
	}
	if len(res.AutoScalingGroups) == 0 {
		return errors.Errorf("not found AutoScalingGroup name: %s", asgName)
	}

	elbSvc := elb.New(sess)
	elbv2Svc := elbv2.New(sess)
	for _, asg := range res.AutoScalingGroups {
		for _, lbName := range asg.LoadBalancerNames {
			log.Printf("[info] deregistering instance %s from %s", instanceID, *lbName)
			_, err := elbSvc.DeregisterInstancesFromLoadBalancer(&elb.DeregisterInstancesFromLoadBalancerInput{
				Instances:        []*elb.Instance{{InstanceId: aws.String(instanceID)}},
				LoadBalancerName: lbName,
			})
			if err != nil {
				return errors.Wrap(err, "failed to DeregisterInstancesFromLoadBalancer")
			}
		}

		for _, tgArn := range asg.TargetGroupARNs {
			log.Printf("[info] deregistering target %s from %s", instanceID, *tgArn)
			_, err := elbv2Svc.DeregisterTargets(&elbv2.DeregisterTargetsInput{
				TargetGroupArn: tgArn,
				Targets:        []*elbv2.TargetDescription{{Id: aws.String(instanceID)}},
			})
			if err != nil {
				return errors.Wrap(err, "failed to DeregisterTargets")
			}
		}
	}
	return nil
}

func complate(svc *autoscaling.AutoScaling, event *events.AutoScalingEvent) error {
	if event.Detail["LifecycleActionToken"] == nil {
		log.Println("[info] skip complete")
		return nil
	}
	_, err := svc.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
		AutoScalingGroupName:  aws.String(str(event.Detail["AutoScalingGroupName"])),
		InstanceId:            aws.String(str(event.Detail["EC2InstanceId"])),
		LifecycleActionResult: aws.String("CONTINUE"),
		LifecycleActionToken:  aws.String(str(event.Detail["LifecycleActionToken"])),
		LifecycleHookName:     aws.String(str(event.Detail["LifecycleHookName"])),
	})
	return err
}

func str(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
