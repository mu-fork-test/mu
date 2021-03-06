package workflows

import (
	"github.com/stelligent/mu/common"
)

// NewEnvironmentTerminator create a new workflow for terminating an environment
func NewEnvironmentTerminator(ctx *common.Context, environmentName string) Executor {

	workflow := new(environmentWorkflow)

	return newWorkflow(
		workflow.environmentServiceTerminator(environmentName, ctx.StackManager, ctx.StackManager, ctx.StackManager),
		workflow.environmentEcsTerminator(environmentName, ctx.StackManager, ctx.StackManager),
		workflow.environmentVpcTerminator(environmentName, ctx.StackManager, ctx.StackManager),
	)
}

func (workflow *environmentWorkflow) environmentServiceTerminator(environmentName string, stackLister common.StackLister, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating Services for environment '%s' ...", environmentName)
		stacks, err := stackLister.ListStacks(common.StackTypeService)
		if err != nil {
			return err
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			err := stackDeleter.DeleteStack(stack.Name)
			if err != nil {
				return err
			}
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			log.Infof("   Undeploying service '%s' from environment '%s'", stack.Tags["service"], environmentName)
			stackWaiter.AwaitFinalStatus(stack.Name)
		}

		return nil
	}
}
func (workflow *environmentWorkflow) environmentEcsTerminator(environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating ECS environment '%s' ...", environmentName)
		envStackName := common.CreateStackName(common.StackTypeCluster, environmentName)
		err := stackDeleter.DeleteStack(envStackName)
		if err != nil {
			return err
		}

		stackWaiter.AwaitFinalStatus(envStackName)
		return nil
	}
}
func (workflow *environmentWorkflow) environmentVpcTerminator(environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating VPC environment '%s' ...", environmentName)
		vpcStackName := common.CreateStackName(common.StackTypeVpc, environmentName)
		err := stackDeleter.DeleteStack(vpcStackName)
		if err != nil {
			log.Debugf("Unable to delete VPC, but ignoring error: %v", err)
		}

		stackWaiter.AwaitFinalStatus(vpcStackName)

		targetStackName := common.CreateStackName(common.StackTypeTarget, environmentName)
		err = stackDeleter.DeleteStack(targetStackName)
		if err != nil {
			log.Debugf("Unable to delete VPC target, but ignoring error: %v", err)
		}

		stackWaiter.AwaitFinalStatus(targetStackName)
		return nil
	}
}
