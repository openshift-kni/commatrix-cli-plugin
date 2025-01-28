package commatrix

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/openshift-kni/commatrix/pkg/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8sapierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/templates"

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
			 kubectl commatrix generate --format (json,yaml,csv,nft)
			
			 # Generate the communication matrix in json format with custom entries:
			 kubectl commatrix generate --format json --customEntriesPath /path/to/customEntriesFile --customEntriesFormat json
			
			 # Generate the communication matrix in json format with debug logs:
			 kubectl commatrix generate --format json --debug
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
	cs                  *client.ClientSet
	configFlags         *genericclioptions.ConfigFlags
	genericiooptions.IOStreams
}

func NewCmd(cs *client.ClientSet, streams genericiooptions.IOStreams) *cobra.Command {
	// Parent command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   "commatrix",
		Short: "Generate an up-to-date communication flows matrix for all ingress flows.",
		Long:  commatrixLong,
	}
	cmds.AddCommand(NewCmdCommatrixGenerate(streams, cs))
	return cmds
}

func NewCommatrixOptions(streams genericiooptions.IOStreams) *CommatrixOptions {
	return &CommatrixOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdAddRoleToUser implements the OpenShift cli add-role-to-user command.
func NewCmdCommatrixGenerate(streams genericiooptions.IOStreams, cs *client.ClientSet) *cobra.Command {
	o := NewCommatrixOptions(streams)
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate an up-to-date communication flows matrix for all ingress flows.",
		Long:    commatrixLong,
		Example: CommatrixExample,
		RunE: func(c *cobra.Command, args []string) (err error) {
			o.cs = cs
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
	if !slices.Contains(validFormats, o.format) {
		return fmt.Errorf("invalid format '%s', valid options are: %s",
			o.format, strings.Join(validFormats, ", "))
	}

	if o.customEntriesPath == "" {
		return nil
	}

	if o.customEntriesFormat == "" {
		return fmt.Errorf("you must specify the --customEntriesFormat when using --customEntriesPath")
	}

	if !slices.Contains(validCustomEntriesFormats, o.customEntriesFormat) {
		return fmt.Errorf("invalid custom entries format '%s', valid options are: %s",
			o.customEntriesFormat, strings.Join(validCustomEntriesFormats, ", "))
	}

	return nil
}

func (o *CommatrixOptions) Run() (err error) {
	if o.debug {
		log.SetLevel(log.DebugLevel)
	}

	utilsHelpers := utils.New(o.cs)

	log.Debug("Detecting deployment and infra types")
	deployment := types.Standard
	infra := types.Cloud
	isOCP := isOpenShift(o.cs)
	if isOCP {
		isSNO, err := utilsHelpers.IsSNOCluster()
		if err != nil {
			return fmt.Errorf("failed to check is sno cluster %s", err)
		}

		if isSNO {
			deployment = types.SNO
		}

		isBM, err := utilsHelpers.IsBMInfra()
		if err != nil {
			return fmt.Errorf("failed to check is bm cluster %s", err)
		}

		if isBM {
			infra = types.Baremetal
		}
	} else {
		isBM, err := IsBMInfra(o.cs)
		if err != nil {
			return fmt.Errorf("failed to check is bm cluster %s", err)
		}

		if isBM {
			infra = types.Baremetal
		}
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

	err = printMatrixByFormat(matrix, o.format, o.Out)
	if err != nil {
		return fmt.Errorf("failed while priting the commatrix %s", err)
	}

	return nil
}

func printMatrixByFormat(matrix *types.ComMatrix, format string, out io.Writer) (err error) {
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
	_, err = out.Write(res)
	if err != nil {
		return fmt.Errorf("error occurred when writing output: %w", err)
	}
	return nil
}

func isOpenShift(cluster *client.ClientSet) bool {
	ctx := context.Background()
	_, err := cluster.Namespaces().Get(ctx, "openshift-kube-apiserver", metav1.GetOptions{})
	return !k8sapierror.IsNotFound(err)
}

func IsBMInfra(cs *client.ClientSet) (bool, error) {
	var nodeList corev1.NodeList
	err := cs.List(context.TODO(), &nodeList)
	if err != nil {
		return false, err
	}

	for _, node := range nodeList.Items {
		labels := node.Labels

		if _, exists := labels["topology.kubernetes.io/region"]; exists {
			return false, nil
		}
		if _, exists := labels["topology.kubernetes.io/zone"]; exists {
			return false, nil
		}
		if _, exists := labels["failure-domain.beta.kubernetes.io/region"]; exists {
			return false, nil
		}
		if _, exists := labels["failure-domain.beta.kubernetes.io/zone"]; exists {
			return false, nil
		}
	}

	return true, nil
}
