package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type EC2InstanceDescription struct {
	Region     string
	InstanceId string
	Tags       map[string]string
}

var (
	stsPresignClient sts.PresignClient
	encDesc          string
	once             sync.Once
)

func MarshalAndEncode(v any) (string, error) {
	jv, err := json.Marshal(v)

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jv), nil
}

func DecodeAndUnmarshal(enc string, v any) error {
	jsonDesc, err := base64.StdEncoding.DecodeString(enc)

	if err != nil {
		return err
	}

	return json.Unmarshal(jsonDesc, v)
}

func AuthenticateInstance(ctx context.Context) (*v4.PresignedHTTPRequest, error) {

	once.Do(func() {
		desc := &EC2InstanceDescription{}

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithEC2IMDSRegion(),
			config.WithEC2RoleCredentialOptions(func(opts *ec2rolecreds.Options) {}))

		if err != nil {
			log.Fatalf("Failed to load default aws config: %v", err)
		}

		log.Println("loaded context")

		err = desc.retrieveIdentity(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve region from ec2 imds: %v", err)
		}

		err = desc.retrieveTags(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve tags from ec2: %v", err)
		}

		encDesc, err = MarshalAndEncode(desc)

		if err != nil {
			log.Fatalf("Failed to marshal instance descritpion: %v", err)
		}

		log.Printf("Encoded description: %v", encDesc)

		stsPresignClient = *sts.NewPresignClient(sts.NewFromConfig(cfg))
	})

	return stsPresignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(opt *sts.PresignOptions) {
		opt.ClientOptions = append(opt.ClientOptions, func(o *sts.Options) {
			o.APIOptions = append(o.APIOptions, smithyhttp.AddHeaderValue("X-Inverting-Proxy-EC2-VM-Desc", encDesc))
		})
	})
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
