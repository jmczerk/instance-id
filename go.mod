module github.com/jmczerk/instance-id

go 1.21.4

require (
	github.com/aws/aws-sdk-go-v2 v1.23.4
	github.com/aws/aws-sdk-go-v2/config v1.25.10
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.8
)

require github.com/jmespath/go-jmespath v0.4.0 // indirect

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.16.8
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.138.1
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.1
	github.com/aws/smithy-go v1.18.1
)
