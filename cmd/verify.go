/*
 * Copyright 2021 Red Hat
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/redhat-certification/chart-verifier/internal/chartverifier/utils"
	apiChecks "github.com/redhat-certification/chart-verifier/pkg/chartverifier/checks"
	apireport "github.com/redhat-certification/chart-verifier/pkg/chartverifier/report"
	apiverifier "github.com/redhat-certification/chart-verifier/pkg/chartverifier/verifier"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	//"helm.sh/helm/v3/pkg/getter"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	//"github.com/redhat-certification/chart-verifier/internal/chartverifier"
)

//func init() {
//	allChecks = chartverifier.DefaultRegistry().AllChecks()
//}

//goland:noinspection GoUnusedGlobalVariable
var (
	// enabledChecksFlag are the checks that should be performed, after the command initialization has happened.
	enabledChecksFlag []string
	// disabledChecksFlag are the checks that should not be performed.
	disabledChecksFlag []string
	// outputFormatFlag contains the output format the user has specified: default, yaml or json.
	outputFormatFlag string
	// setOverridesFlag contains the overrides the user has specified through the --set flag.
	setOverridesFlag []string
	// openshiftVersionFlag set the value of `certifiedOpenShiftVersions` in the report
	openshiftVersionFlag string
	// write report to file
	reportToFile bool
	// write an error log file
	suppressErrorLog bool
	// provider controls chart deliver mechanism.
	providerDelivery bool
	//client timeout
	clientTimeout time.Duration
)

func buildChecks(enabled []string, unEnabled []string) ([]apiChecks.CheckName, []apiChecks.CheckName, error) {

	var enabledChecks []apiChecks.CheckName
	var unEnabledChecks []apiChecks.CheckName
	var convertErr error
	if len(enabled) > 0 && len(unEnabled) > 0 {
		return enabledChecks, unEnabledChecks, errors.New("--enable and --disable can't be used at the same time")
	} else if len(enabled) > 0 {
		enabledChecks, convertErr = convertChecks(enabled)
		if convertErr != nil {
			return enabledChecks, unEnabledChecks, convertErr
		}
	} else if len(unEnabled) > 0 {
		unEnabledChecks, convertErr = convertChecks(unEnabled)
		if convertErr != nil {
			return enabledChecks, unEnabledChecks, convertErr
		}
	}
	return enabledChecks, unEnabledChecks, nil

}

func convertChecks(checks []string) ([]apiChecks.CheckName, error) {
	var apiCheckSet []apiChecks.CheckName
	for _, check := range checks {
		checkFound := false
		for _, checkName := range apiChecks.GetChecks() {
			if apiChecks.CheckName(check) == checkName {
				apiCheckSet = append(apiCheckSet, checkName)
				checkFound = true
			}
		}
		if !checkFound {
			return apiCheckSet, errors.New(fmt.Sprintf("enabled check is invalid :%s", check))
		}
	}
	return apiCheckSet, nil

}

func convertToMap(values []string) map[string]interface{} {
	valueMap := make(map[string]interface{})
	for _, val := range values {
		parts := strings.Split(val, "=")
		valueMap[strings.ToLower(parts[0])] = parts[1]
	}
	return valueMap
}

// settings comes from Helm, to extract the same configuration values Helm uses.
var settings = cli.New()

type verifyOptions struct {
	ValueFiles []string
	Values     []string
}

// NewVerifyCmd creates ...
func NewVerifyCmd(config *viper.Viper) *cobra.Command {

	// opts contains command line options extracted from the environment.
	opts := &values.Options{}

	// verifyOpts contains this specific command options.
	verifyOpts := &verifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify <chart-uri>",
		Args:  cobra.ExactArgs(1),
		Short: "Verifies a Helm chart by checking some of its characteristics",
		RunE: func(cmd *cobra.Command, args []string) error {

			reportFormat := apireport.YamlReport
			if outputFormatFlag == "json" {
				reportFormat = apireport.JsonReport
			}

			reportName := ""
			if reportToFile {
				if outputFormatFlag == "json" {
					reportName = "report.json"
				} else {
					reportName = "report.yaml"
				}
			}

			enabledChecks, unEnabledChecks, checksErr := buildChecks(enabledChecksFlag, disabledChecksFlag)
			if checksErr != nil {
				return checksErr
			}

			utils.InitLog(cmd, reportName, suppressErrorLog)

			utils.LogInfo(fmt.Sprintf("Chart Verifer %s.", Version))
			utils.LogInfo(fmt.Sprintf("Verify : %s", args[0]))
			utils.LogInfo(fmt.Sprintf("Client timeout: %s", clientTimeout))

			valueMap := convertToMap(verifyOpts.Values)
			for key, val := range viper.AllSettings() {
				valueMap[strings.ToLower(key)] = val
			}

			verifier := apiverifier.NewVerifier()

			if len(enabledChecks) > 0 {
				verifier = verifier.EnableChecks(enabledChecks)
			} else if len(unEnabledChecks) > 0 {
				verifier = verifier.UnEnableChecks(unEnabledChecks)
			}

			var runErr error
			verifier, runErr = verifier.SetBoolean(apiverifier.ProviderDelivery, providerDelivery).
				SetBoolean(apiverifier.SuppressErrorLog, suppressErrorLog).
				SetDuration(apiverifier.Timeout, clientTimeout).
				SetString(apiverifier.OpenshiftVersion, []string{openshiftVersionFlag}).
				SetString(apiverifier.ChartValues, opts.ValueFiles).
				SetString(apiverifier.KubeApiServer, []string{settings.KubeAPIServer}).
				SetString(apiverifier.KubeAsUser, []string{settings.KubeAsUser}).
				SetString(apiverifier.KubeCaFile, []string{settings.KubeCaFile}).
				SetString(apiverifier.KubeConfig, []string{settings.KubeConfig}).
				SetString(apiverifier.KubeContext, []string{settings.KubeContext}).
				SetString(apiverifier.Namespace, []string{settings.Namespace()}).
				SetString(apiverifier.KubeApiServer, []string{settings.KubeAPIServer}).
				SetString(apiverifier.RegistryConfig, []string{settings.RegistryConfig}).
				SetString(apiverifier.RepositoryConfig, []string{settings.RepositoryConfig}).
				SetString(apiverifier.RepositoryCache, []string{settings.RepositoryCache}).
				SetString(apiverifier.KubeAsGroups, settings.KubeAsGroups).
				SetValues(apiverifier.CommandSet, valueMap).
				SetValues(apiverifier.ChartSet, convertToMap(opts.Values)).
				SetValues(apiverifier.ChartSetFile, convertToMap(opts.FileValues)).
				SetValues(apiverifier.ChartSetString, convertToMap(opts.StringValues)).
				Run(args[0])

			if runErr != nil {
				return runErr
			}

			report, reportErr := verifier.GetReport().GetContent(reportFormat)
			if reportErr != nil {
				return reportErr
			}

			utils.WriteStdOut(report)

			utils.WriteLogs(outputFormatFlag)

			return nil
		},
	}

	settings.AddFlags(cmd.Flags())

	cmd.Flags().StringSliceVarP(&opts.ValueFiles, "chart-values", "F", nil, "specify values in a YAML file or a URL (can specify multiple)")

	cmd.Flags().StringSliceVarP(&opts.Values, "chart-set", "S", nil, "set values for the chart (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	cmd.Flags().StringSliceVarP(&opts.StringValues, "chart-set-string", "X", nil, "set STRING values for the chart (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	cmd.Flags().StringSliceVarP(&opts.FileValues, "chart-set-file", "G", nil, "set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")

	cmd.Flags().StringSliceVarP(&enabledChecksFlag, "enable", "e", nil, "only the informed checks will be enabled")

	cmd.Flags().StringSliceVarP(&disabledChecksFlag, "disable", "x", nil, "all checks will be enabled except the informed ones")

	cmd.Flags().StringVarP(&outputFormatFlag, "output", "o", "", "the output format: default, json or yaml")

	cmd.Flags().StringSliceVarP(&verifyOpts.Values, "set", "s", []string{}, "overrides a configuration, e.g: dummy.ok=false")

	cmd.Flags().StringSliceVarP(&verifyOpts.ValueFiles, "set-values", "f", nil, "specify application and check configuration values in a YAML file or a URL (can specify multiple)")
	cmd.Flags().StringVarP(&openshiftVersionFlag, "openshift-version", "V", "", "version of OpenShift used in the cluster")
	cmd.Flags().DurationVar(&clientTimeout, "timeout", 30*time.Minute, "time to wait for completion of chart install and test")
	cmd.Flags().BoolVarP(&reportToFile, "write-to-file", "w", false, "write report to ./chartverifier/report.yaml (default: stdout)")
	cmd.Flags().BoolVarP(&suppressErrorLog, "suppress-error-log", "E", false, "suppress the error log (default: written to ./chartverifier/verifier-<timestamp>.log)")
	cmd.Flags().BoolVarP(&providerDelivery, "provider-delivery", "d", false, "chart provider will provide the chart delivery mechanism (default: false)")
	return cmd
}

func init() {
	rootCmd.AddCommand(NewVerifyCmd(viper.GetViper()))
}
