package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

type EC2InstanceIdentity struct {
	Description EC2InstanceDescription
	Expiration  time.Time
}

type PresignedEC2InstanceIdentityRequest struct {
	URL          url.URL
	Method       string
	SignedHeader http.Header
}

func (r *PresignedEC2InstanceIdentityRequest) MarshalAndEncode() (string, error) {
	j, err := json.Marshal(r)

	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(j), nil
}

func DecodeAndUnmarshal(enc string) (*PresignedEC2InstanceIdentityRequest, error) {
	j, err := base64.URLEncoding.DecodeString(enc)

	if err != nil {
		return nil, err
	}

	req := &PresignedEC2InstanceIdentityRequest{}

	err = json.Unmarshal(j, req)

	if err != nil {
		return nil, err
	}

	return req, nil
}

// ec2InstanceDescriptionSingleton is a singleton wrapper around EC2InstanceDescription.
type ec2InstanceDescriptionSingleton struct {
	description EC2InstanceDescription
	once        sync.Once
}

// stsPresignClientSingleton is a singleton wrapper around sts.PresignClient.
type stsPresignClientSingleton struct {
	client sts.PresignClient
	once   sync.Once
}

type cachedIdentity struct {
	instanceIdentity        EC2InstanceIdentity
	encodedInstanceIdentity string
	mutex                   sync.Mutex
}

var (
	// description is a singleton EC2InstanceDescription.  This allows us to interogate EC2
	// Instance Metadata Service (IMDS) once at process startup, as these are static values
	description ec2InstanceDescriptionSingleton

	// stsPresignClient is a singleton sts.PresignClient instance.  This allows us to to set up a
	// long-lived client with an aws.CredentialsCache object which will take care of concurrency-
	// safe caching and retrieval of credentials.
	stsPresignClient stsPresignClientSingleton

	cachedId cachedIdentity
)

func AuthenticateInstance(ctx context.Context) (*PresignedEC2InstanceIdentityRequest, error) {

	encDesc, err := cachedId.getOrRefreshEncodedIdentity()

	if err != nil {
		return nil, err
	}

	awsReq, err := stsPresignClient.getClient(ctx).PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(opt *sts.PresignOptions) {
		opt.ClientOptions = append(opt.ClientOptions, func(opt *sts.Options) {
			opt.APIOptions = append(opt.APIOptions, smithyhttp.AddHeaderValue("X-Inverting-Proxy-EC2-VM-Desc", encDesc))
		})
	})

	if err != nil {
		return nil, err
	}

	url, err := url.Parse(awsReq.URL)

	if err != nil {
		return nil, err
	}

	req := &PresignedEC2InstanceIdentityRequest{
		Method:       awsReq.Method,
		URL:          *url,
		SignedHeader: awsReq.SignedHeader,
	}

	return req, nil
}

func (d *ec2InstanceDescriptionSingleton) getDescription(ctx context.Context) *EC2InstanceDescription {
	d.once.Do(func() {
		cfg := loadConfig(ctx)
		err := d.description.retrieveIdentity(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve region from ec2 imds: %v", err)
		}

		err = d.description.retrieveTags(ctx, &cfg)

		if err != nil {
			log.Fatalf("Failed to retrieve tags from ec2: %v", err)
		}
	})

	return &d.description
}

func (c *stsPresignClientSingleton) getClient(ctx context.Context) *sts.PresignClient {
	c.once.Do(func() {
		cfg := loadConfig(ctx)
		c.client = *sts.NewPresignClient(sts.NewFromConfig(cfg))
	})

	return &c.client
}

func loadConfig(ctx context.Context) aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
		config.WithEC2RoleCredentialOptions(func(opts *ec2rolecreds.Options) {}))

	if err != nil {
		log.Fatalf("Failed to load default aws config: %v", err)
	}

	return cfg
}

func (id *cachedIdentity) getOrRefreshEncodedIdentity() (string, error) {

	// Get system clock time before taking the lock
	now := time.Now()
	adj := now.Add(time.Second * 120)

	id.mutex.Lock()
	defer id.mutex.Unlock()

	if adj.After(id.instanceIdentity.Expiration) {

		// This is expected to be the signicifantly less frequent path, as expiration window is
		// expected to be on the order of minutes.  Thus the intentional choice to do the marshal
		// and encode under lock, so that we can avoid these cycles in the common case.

		id.instanceIdentity.Expiration = now.Add(time.Minute * 15)
		j, err := json.Marshal(id.instanceIdentity)

		if err != nil {
			return "", err
		}

		id.encodedInstanceIdentity = base64.URLEncoding.EncodeToString(j)
	}

	return id.encodedInstanceIdentity, nil
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
