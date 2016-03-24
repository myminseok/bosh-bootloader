package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/pivotal-cf-experimental/bosh-bootloader/application"
	"github.com/pivotal-cf-experimental/bosh-bootloader/aws"
	"github.com/pivotal-cf-experimental/bosh-bootloader/aws/cloudformation"
	"github.com/pivotal-cf-experimental/bosh-bootloader/aws/cloudformation/templates"
	"github.com/pivotal-cf-experimental/bosh-bootloader/aws/ec2"
	"github.com/pivotal-cf-experimental/bosh-bootloader/bosh"
	"github.com/pivotal-cf-experimental/bosh-bootloader/boshinit"
	"github.com/pivotal-cf-experimental/bosh-bootloader/boshinit/manifests"
	"github.com/pivotal-cf-experimental/bosh-bootloader/commands"
	"github.com/pivotal-cf-experimental/bosh-bootloader/commands/unsupported"
	"github.com/pivotal-cf-experimental/bosh-bootloader/helpers"
	"github.com/pivotal-cf-experimental/bosh-bootloader/ssl"
	"github.com/pivotal-cf-experimental/bosh-bootloader/storage"
)

func main() {
	uuidGenerator := helpers.NewUUIDGenerator(rand.Reader)
	stringGenerator := helpers.NewStringGenerator(rand.Reader)
	logger := application.NewLogger(os.Stdout)

	keyPairCreator := ec2.NewKeyPairCreator(uuidGenerator)
	keyPairChecker := ec2.NewKeyPairChecker()
	keyPairManager := ec2.NewKeyPairManager(keyPairCreator, keyPairChecker, logger)
	keyPairSynchronizer := ec2.NewKeyPairSynchronizer(keyPairManager)
	stateStore := storage.NewStore()
	templateBuilder := templates.NewTemplateBuilder(logger)
	stackManager := cloudformation.NewStackManager(logger)
	infrastructureCreator := cloudformation.NewInfrastructureCreator(templateBuilder, stackManager)
	awsClientProvider := aws.NewClientProvider()
	sslKeyPairGenerator := ssl.NewKeyPairGenerator(time.Now, rsa.GenerateKey, x509.CreateCertificate)

	tempDir, err := ioutil.TempDir("", "bosh-init")
	if err != nil {
		fail(err)
	}

	boshInitPath, err := exec.LookPath("bosh-init")
	if err != nil {
		fail(err)
	}

	boshInitCommandBuilder := boshinit.NewCommandBuilder(boshInitPath, tempDir, os.Stdout, os.Stderr)
	boshInitDeployCommand := boshInitCommandBuilder.DeployCommand()
	cloudProviderManifestBuilder := manifests.NewCloudProviderManifestBuilder(stringGenerator)
	jobsManifestBuilder := manifests.NewJobsManifestBuilder(stringGenerator)
	boshInitManifestBuilder := manifests.NewManifestBuilder(logger, sslKeyPairGenerator, stringGenerator, cloudProviderManifestBuilder, jobsManifestBuilder)
	boshInitRunner := boshinit.NewRunner(tempDir, boshInitDeployCommand, logger)
	boshDeployer := boshinit.NewBOSHDeployer(boshInitManifestBuilder, boshInitRunner, logger)
	boshCloudConfigGenerator := bosh.NewCloudConfigGenerator()
	cloudConfigurator := bosh.NewCloudConfigurator(logger, boshCloudConfigGenerator)
	availabilityZoneRetriever := ec2.NewAvailabilityZoneRetriever()

	app := application.New(application.CommandSet{
		"help":    commands.NewUsage(os.Stdout),
		"version": commands.NewVersion(os.Stdout),
		"unsupported-deploy-bosh-on-aws-for-concourse": unsupported.NewDeployBOSHOnAWSForConcourse(infrastructureCreator, keyPairSynchronizer, awsClientProvider, boshDeployer, stringGenerator, cloudConfigurator, availabilityZoneRetriever),
	}, stateStore, commands.NewUsage(os.Stdout).Print)

	err = app.Run(os.Args[1:])
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n\n%s\n", err)
	os.Exit(1)
}
