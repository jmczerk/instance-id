package utils

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type EC2InstanceDescription struct {
	Region     string
	InstanceId string
	Tags       map[string]string
}

type EC2InstanceAuthenticator struct {
	EC2InstanceDescription
	v4.PresignedHTTPRequest
}

var (
	stsPresignClient sts.PresignClient
	description      EC2InstanceDescription
	once             sync.Once
)

func AuthenticateInstance(ctx context.Context) (EC2InstanceAuthenticator, error) {

	once.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)

		if err != nil {
			log.Fatalf("Failed to load default aws config: %v", err)
		}

		log.Println("loaded context")

		err = description.retrieveIdentity(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve region from ec2 imds: %v", err)
		}

		err = description.retrieveTags(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve tags from ec2: %v", err)
		}

		stsPresignClient = *sts.NewPresignClient(sts.NewFromConfig(cfg))
	})

	req, err := stsPresignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

	if err != nil {
		return EC2InstanceAuthenticator{}, err
	}

	stsPresignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return EC2InstanceAuthenticator{
		EC2InstanceDescription: description,
		PresignedHTTPRequest:   *req,
	}, nil
}

func (desc *EC2InstanceDescription) retrieveIdentity(ctx context.Context, cfg *aws.Config) error {
	client := imds.NewFromConfig(*cfg)
	idRsp, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})

	if err != nil {
		return err
	}

	desc.Region = idRsp.Region
	desc.InstanceId = idRsp.InstanceID
	return nil
}

func (desc *EC2InstanceDescription) retrieveTags(ctx context.Context, cfg *aws.Config) error {
	client := ec2.NewFromConfig(*cfg)

	rsp, err := client.DescribeTags(ctx, &ec2.DescribeTagsInput{
		Filters: []ec2_types.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []string{
					desc.InstanceId,
				},
			},
			{
				Name: aws.String("resource-type"),
				Values: []string{
					"instance",
				},
			},
		},
	})

	if err != nil {
		return err
	}

	desc.Tags = make(map[string]string)
	for _, tag := range rsp.Tags {
		desc.Tags[*tag.Key] = *tag.Value
	}
	return nil
}
