package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/openshift-kni/commatrix/pkg/client"
	commatrixcreator "github.com/openshift-kni/commatrix/pkg/commatrix-creator"
	"github.com/openshift-kni/commatrix/pkg/endpointslices"
	"github.com/openshift-kni/commatrix/pkg/types"
	"github.com/openshift-kni/commatrix/pkg/utils"
)

var (
	commatrixLong = templates.LongDesc(`
		Generate the communication matrix \n
		This command to generate the communication matrix on nodes`)
	CommatrixExample = templates.Examples(`
		oc commatrix generate
	`)
)

type CommatrixOptions struct {
	destDir             string
	format              string
	customEntriesPath   string
	customEntriesFormat string
	debug               bool
	configFlags         *genericclioptions.ConfigFlags

	genericiooptions.IOStreams
}

func NewCmdCommatrix(streams genericiooptions.IOStreams) *cobra.Command {
	// Parent command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   "commatrix",
		Short: "Generate the communication matrix",
		Long:  commatrixLong,
	}
	cmds.AddCommand(NewCmdCommatrixGenerate(streams))

	return cmds
}

func Newcommatrix(streams genericiooptions.IOStreams) *CommatrixOptions {
	return &CommatrixOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdAddRoleToUser implements the OpenShift cli add-role-to-user command.
func NewCmdCommatrixGenerate(streams genericiooptions.IOStreams) *cobra.Command {
	o := Newcommatrix(streams)
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate the communication matrix",
		Long:    commatrixLong,
		Example: CommatrixExample,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&o.destDir, "destDir", "communication-matrix", "Output files dir")
	cmd.Flags().StringVar(&o.format, "format", "csv", "Desired format (json,yaml,csv,nft)")
	cmd.Flags().StringVar(&o.customEntriesPath, "customEntriesPath", "", "Add custom entries from a file to the matrix")
	cmd.Flags().StringVar(&o.customEntriesFormat, "customEntriesFormat", "", "Set the format of the custom entries file (json,yaml,csv)")
	cmd.Flags().BoolVar(&o.debug, "debug", false, "Debug logs")
	return cmd
}

// Complete initializes the options based on the provided arguments and flags.
func (o *CommatrixOptions) Complete(cmd *cobra.Command, args []string) error {
	// Validate the number of arguments
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}

	// Initialize any dependencies or derived fields if needed.
	if o.destDir == "" {
		o.destDir = "communication-matrix" // Default value
	}

	if o.format == "" {
		o.format = "csv" // Default format
	}

	if o.customEntriesPath != "" && o.customEntriesFormat == "" {
		return fmt.Errorf("you must specify the --customEntriesFormat when using --customEntriesPath")
	}

	return nil
}

func (o *CommatrixOptions) Validate() error {
	// Validate destination directory.
	if o.destDir == "" {
		return fmt.Errorf("destination directory cannot be empty")
	}

	// Validate format
	validFormats := map[string]bool{"csv": true, "json": true, "yaml": true, "nft": true}
	if _, valid := validFormats[o.format]; !valid {
		return fmt.Errorf("invalid format '%s', valid options are: csv, json, yaml, nft", o.format)
	}

	// Validate custom entries path and format.
	if o.customEntriesPath != "" {
		if o.customEntriesFormat == "" {
			return fmt.Errorf("you must specify the --customEntriesFormat when using --customEntriesPath")
		}

		validCustomFormats := map[string]bool{"csv": true, "json": true, "yaml": true}
		if _, valid := validCustomFormats[o.customEntriesFormat]; !valid {
			return fmt.Errorf("invalid custom entries format '%s', valid options are: csv, json, yaml", o.customEntriesFormat)
		}
	}

	return nil
}

func (o *CommatrixOptions) Run() error {
	if o.debug {
		log.SetLevel(log.DebugLevel)
	}

	cs, err := client.New()
	if err != nil {
		return fmt.Errorf("%s: %v", "Failed creating the k8s client", err)
	}

	utilsHelpers := utils.New(cs)
	log.Debug("Utils helpers initialized")

	deployment, infra, err := detectDeploymentAndInfra(utilsHelpers)
	if err != nil {
		return err
	}

	epExporter, err := endpointslices.New(cs)
	if err != nil {
		return fmt.Errorf("failed creating the endpointslices exporter %s", err)
	}

	matrix, err := generateCommunicationMatrix(epExporter, deployment, infra, o.customEntriesPath, o.customEntriesFormat)
	if err != nil {
		return err
	}

	var res []byte
	switch o.format {
	case "json":
		res, err = matrix.ToJSON()
		if err != nil {
			return err
		}
	case "csv":
		res, err = matrix.ToCSV()
		if err != nil {
			return err
		}
	case "yaml":
		res, err = matrix.ToYAML()
		if err != nil {
			return err
		}
	case "nft":
		res, err = matrix.ToNFTables()
		if err != nil {
			return err
		}
	}

	fmt.Println(string(res))

	return nil
}

func detectDeploymentAndInfra(utilsHelpers utils.UtilsInterface) (types.Deployment, types.Env, error) {
	log.Debug("Detecting deployment and infra types")

	deployment := types.Standard
	isSNO, err := utilsHelpers.IsSNOCluster()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to detect deployment type %s", err)
	}

	if isSNO {
		deployment = types.SNO
	}

	infra := types.Cloud
	isBM, err := utilsHelpers.IsBMInfra()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to detect infra type %s", err)
	}
	if isBM {
		infra = types.Baremetal
	}

	return deployment, infra, err
}

func generateCommunicationMatrix(epExporter *endpointslices.EndpointSlicesExporter, deployment types.Deployment, infra types.Env, customEntriesPath, customEntriesFormat string) (*types.ComMatrix, error) {
	log.Debug("Creating communication matrix")
	commMatrix, err := commatrixcreator.New(epExporter, customEntriesPath, customEntriesFormat, infra, deployment)
	if err != nil {
		return nil, err
	}

	matrix, err := commMatrix.CreateEndpointMatrix()
	if err != nil {
		return nil, err
	}

	return matrix, nil
}
