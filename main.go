// https://github.com/pulumi/automation-api-examples/blob/main/go/pulumi_over_http/main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var projectName = "pulumi-daemon"

func PulumiProgram() pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		// our program defines a s3 website.
		// here we create the bucket
		bucket, err := s3.NewBucket(ctx, "demo", nil)
		if err != nil {
			return err
		}

		// export the website URL
		ctx.Export("bucketName", bucket.Bucket)
		return nil
	}
}

func main() {
	ctx := context.Background()

	stackName := "dev"

	program := PulumiProgram()

	s, err := auto.SelectStackInlineSource(ctx, stackName, projectName, program)
	if err != nil {
		if auto.IsSelectStack404Error(err) {
			fmt.Printf("Stack not found, trying to create it: %s\n", err.Error())
			s, err = auto.NewStackInlineSource(ctx, stackName, projectName, program)
			if err != nil {
				// if stack already exists, 409
				if auto.IsCreateStack409Error(err) {
					fmt.Printf("Error 404 creating stack: %s\n", err.Error())
					return
				}
				fmt.Printf("Error creating stack: %s\n", err.Error())
				return
			}
		} else {
			fmt.Printf("Error selecting stack: %s\n", err.Error())
			return
		}
	}

	fmt.Printf("Selected stack %q\n", stackName)

	// set stack configuration specifying the AWS region to deploy
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-east-1"})
	s.SetConfig(ctx, "aws:accessKey", auto.ConfigValue{Value: "test"})
	s.SetConfig(ctx, "aws:secretKey", auto.ConfigValue{Value: "test"})
	s.SetConfig(ctx, "aws:s3_force_path_style", auto.ConfigValue{Value: "true"})
	s.SetConfig(ctx, "aws:skipCredentialsValidation", auto.ConfigValue{Value: "true"})
	s.SetConfig(ctx, "aws:skipRequestingAccountId", auto.ConfigValue{Value: "true"})

	s.SetConfig(ctx, "aws:endpoints", auto.ConfigValue{Value: "[{\"s3\": \"http://localhost:4566\"}]"})

	cm, err := s.GetAllConfig(ctx)

	fmt.Printf("Config: %+v\n", cm)

	fmt.Println("Stack configured")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refresh succeeded!")

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our fargate web service
	stackResult, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bucket name: %s\n", stackResult.Outputs["bucketName"].Value.(string))

	fmt.Println("Update succeeded!")

}
