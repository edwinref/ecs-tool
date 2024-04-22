package lib

import (
    "context"
    "fmt"
    "github.com/apex/log"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    "github.com/fujiwara/ecsta"
)

var sessionInstance *ecs.Client
var sessionConfig aws.Config // Variable for session configuration

// InitAWS initializes a new AWS session with the specified profile
func InitAWS(profile string) error {
    if sessionInstance == nil {
        cfg, err := config.LoadDefaultConfig(context.TODO(),
            config.WithSharedConfigProfile(profile),
        )
        if err != nil {
            return fmt.Errorf("failed to load configuration: %w", err)
        }
        sessionInstance = ecs.NewFromConfig(cfg)
        sessionConfig = cfg // Save session configuration
    }
    return nil
}

// ExecFargate executes a command in a specified container on an ECS Fargate service
func ExecFargate(profile, cluster, service, containerName, command string) error {
    if err := InitAWS(profile); err != nil {
        return fmt.Errorf("failed to initialize AWS session: %w", err)
    }

    region := sessionConfig.Region // Use the saved region from session configuration
    ecstaApp, err := ecsta.New(context.TODO(), region, cluster)
    if err != nil {
        return fmt.Errorf("failed to create ecsta application: %w", err)
    }
    service = "app"

    entrypoint := "/usr/bin/ssm-parent"
    configPath := "/app/.ssm-parent.yaml"
    fullCommand := fmt.Sprintf("%s -c %s run -- %s", entrypoint, configPath, command)
    execOpt := ecsta.ExecOption{
        Service:   aws.String(service),
        Container: containerName,
        Command:   fullCommand,
    }

    if err := ecstaApp.RunExec(context.Background(), &execOpt); err != nil {
        return fmt.Errorf("failed to execute command: %w", err)
    }

    log.Info("Command executed successfully")
    return nil
}

// FindLatestTaskArn locates the most recent task ARN for a specified ECS service
func FindLatestTaskArn(client *ecs.Client, clusterName, serviceName string) (string, error) {
    resp, err := client.ListTasks(context.TODO(), &ecs.ListTasksInput{
        Cluster:     aws.String(clusterName),
        ServiceName: aws.String(serviceName),
    })
    if err != nil {
        return "", fmt.Errorf("error listing tasks: %w", err)
    }
    if len(resp.TaskArns) == 0 {
        return "", fmt.Errorf("no tasks found for service %s on cluster %s", serviceName, clusterName)
    }

    return resp.TaskArns[0], nil
}