package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func main() {
	ctx := context.Background()

	src := "docker://alpine@sha256:48d9183eb12a05c99bcc0bf44a003607b8e941e1d4f41f9ad12bdcc4b5672f86"
	dest := "docker://someendpoint/alpine"

	policyContext, policyContextErr := signature.NewPolicyContext(
		&signature.Policy{
			Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
		},
	)
	if policyContextErr != nil {
		log.Fatal(fmt.Errorf("error loading trust policy: %w", policyContextErr))
	}
	defer func() {
		if err := policyContext.Destroy(); err != nil {
			// handle error better?
			log.Printf("failure tearing down policy context, %v", err)

		}
	}()

	srcRef, srcRefErr := alltransports.ParseImageName(src)
	if srcRefErr != nil {
		log.Fatal(fmt.Errorf("invalid source name %s, %w", src, srcRefErr))
	}

	destRef, destRefErr := alltransports.ParseImageName(dest)
	if destRefErr != nil {
		log.Fatal(fmt.Errorf("invalid destination name %s, %w", dest, destRefErr))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("default"), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ecrClient := ecr.NewFromConfig(cfg)

	r, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		log.Fatal(err)
	}

	v, err := base64.StdEncoding.DecodeString(aws.ToString(r.AuthorizationData[0].AuthorizationToken))
	if err != nil {
		log.Fatal(err)
	}

	parts := strings.Split(string(v), ":")

	ac := &types.DockerAuthConfig{
		Username: parts[0],
		Password: parts[1],
	}

	destCtx := &types.SystemContext{
		DockerAuthConfig: ac,
	}

	result, err := copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		RemoveSignatures:                      false,
		Signers:                               nil,
		SignBy:                                "",
		SignPassphrase:                        "",
		SignBySigstorePrivateKeyFile:          "",
		SignSigstorePrivateKeyPassphrase:      nil,
		SignIdentity:                          nil,
		ReportWriter:                          nil,
		SourceCtx:                             nil,
		DestinationCtx:                        destCtx,
		ProgressInterval:                      0,
		Progress:                              nil,
		PreserveDigests:                       false,
		ForceManifestMIMEType:                 "",
		ImageListSelection:                    0,
		Instances:                             nil,
		PreferGzipInstances:                   0,
		OciEncryptConfig:                      nil,
		OciEncryptLayers:                      nil,
		OciDecryptConfig:                      nil,
		ConcurrentBlobCopiesSemaphore:         nil,
		MaxParallelDownloads:                  0,
		OptimizeDestinationImageAlreadyExists: false,
		DownloadForeignLayers:                 false,
		EnsureCompressionVariantsExist:        nil,
		ForceCompressionFormat:                false,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(result))
}
