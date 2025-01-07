package commatrix

import (
	"fmt"
	"strings"

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
			  Generate an up-to-date communication flows matrix for all ingress flows of OpenShift (multi-node and single-node deployments) and Operators.

			  For additional details, please refer to the communication matrix repo(https://github.com/openshift-kni/commatrix/blob/main/README.md)

	`)
	CommatrixExample = templates.Examples(`
			 # Generate the communication matrix in chosen format:
			 oc commatrix generate --format (json,yaml,csv,nft)
			
			 # Generate the communication matrix in json format with custom entries:
			 oc commatrix generate --format json --customEntriesPath /path/to/customEntriesFile --customEntriesFormat json
			
			 # Generate the communication matrix in json format with debug logs:
			 oc commatrix generate --format json --debug
	`)
)

var (
	validFormats = []string{
		types.FormatCSV,
		types.FormatJSON,
		types.FormatYAML,
		types.FormatNFT,
	}

	validCustomEntriesFormats = []string{
		types.FormatCSV,
		types.FormatJSON,
		types.FormatYAML,
	}
)

type CommatrixOptions struct {
	format              string
	customEntriesPath   string
	customEntriesFormat string
	debug               bool
	configFlags         *genericclioptions.ConfigFlags
	cs                  *client.ClientSet
	utilsHelpers        utils.UtilsInterface
	genericiooptions.IOStreams
}

func NewCmd(streams genericiooptions.IOStreams) *cobra.Command {
	// Parent command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   "commatrix",
		Short: "Generate an up-to-date communication flows matrix for all ingress flows.",
		Long:  commatrixLong,
	}
	cmds.AddCommand(NewCmdCommatrixGenerate(streams))

	return cmds
}

func NewcommatrixOp(streams genericiooptions.IOStreams) *CommatrixOptions {
	return &CommatrixOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdAddRoleToUser implements the OpenShift cli add-role-to-user command.
func NewCmdCommatrixGenerate(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewcommatrixOp(streams)
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate an up-to-date communication flows matrix for all ingress flows.",
		Long:    commatrixLong,
		Example: CommatrixExample,
		RunE: func(c *cobra.Command, args []string) (err error) {
			o.cs, err = client.New()
			if err != nil {
				return fmt.Errorf("%s: %v", "failed to create the k8s client", err)
			}

			o.utilsHelpers = utils.New(o.cs)
			if err = o.Complete(c, args); err != nil {
				return err
			}

			if err = o.Validate(); err != nil {
				return err
			}

			if err = o.Run(); err != nil {
				return err
			}

			return nil
		},
	}
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

	return nil
}

func (o *CommatrixOptions) Validate() error {
	if !isValidFormat(o.format, validFormats) {
		return fmt.Errorf("invalid format '%s', valid options are: %s",
			o.format, strings.Join(validFormats, ", "))
	}

	if o.customEntriesPath != "" {
		if o.customEntriesFormat == "" {
			return fmt.Errorf("you must specify the --customEntriesFormat when using --customEntriesPath")
		}

		if !isValidFormat(o.customEntriesFormat, validCustomEntriesFormats) {
			return fmt.Errorf("invalid custom entries format '%s', valid options are: %s",
				o.customEntriesFormat, strings.Join(validCustomEntriesFormats, ", "))
		}
	}

	return nil
}

func isValidFormat(format string, validFormats []string) bool {
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

func (o *CommatrixOptions) Run() (err error) {
	if o.debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("Detecting deployment and infra types")
	deployment := types.Standard
	isSNO, err := o.utilsHelpers.IsSNOCluster()
	if err != nil {
		return fmt.Errorf("failed to check is sno cluster %s", err)
	}

	if isSNO {
		deployment = types.SNO
	}

	infra := types.Cloud
	isBM, err := o.utilsHelpers.IsBMInfra()
	if err != nil {
		return fmt.Errorf("failed to check is bm cluster %s", err)
	}

	if isBM {
		infra = types.Baremetal
	}

	epExporter, err := endpointslices.New(o.cs)
	if err != nil {
		return fmt.Errorf("failed creating the endpointslices exporter %s", err)
	}

	log.Debug("Creating communication matrix")
	commMatrix, err := commatrixcreator.New(epExporter, o.customEntriesPath, o.customEntriesFormat, infra, deployment)
	if err != nil {
		return err
	}

	matrix, err := commMatrix.CreateEndpointMatrix()
	if err != nil {
		return err
	}

	err = printMatrixByFormat(matrix, o.format)
	if err != nil {
		return fmt.Errorf("failed while priting the commatrix %s", err)
	}

	return nil
}

func printMatrixByFormat(matrix *types.ComMatrix, format string) (err error) {
	var res []byte
	switch format {
	case types.FormatJSON:
		res, err = matrix.ToJSON()
		if err != nil {
			return err
		}
	case types.FormatCSV:
		res, err = matrix.ToCSV()
		if err != nil {
			return err
		}
	case types.FormatYAML:
		res, err = matrix.ToYAML()
		if err != nil {
			return err
		}
	case types.FormatNFT:
		res, err = matrix.ToNFTables()
		if err != nil {
			return err
		}
	}

	fmt.Println(string(res))
	return nil
}
