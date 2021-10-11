# asg-lifecycle-hook-ec2

## DESCRIPTION

asg-lifecycle-hook-ec2 is a AWS Lambda function to drain EC2 instances when the instances will be terminated by AutoScalingGroup(ASG).

This function is supposed to be called by ASG lifecycle hook.

## Install

```console
docker pull ghcr.io/kayac/asg-lifecycle-hook-ec2:v0.0.3
```

[Release packages](https://github.com/kayac/asg-lifecycle-hook-ec2/releases)

## Configuration

All configuration parameters are defined in environment variables.

- `WAIT_SECONDS`: Seconds until complete the lifecycle action after the instance deregistered from load balancers.

## LICENSE

MIT
