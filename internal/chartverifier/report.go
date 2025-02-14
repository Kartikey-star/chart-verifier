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

package chartverifier

import (
	"fmt"
	"github.com/redhat-certification/chart-verifier/internal/chartverifier/checks"
	apiChecks "github.com/redhat-certification/chart-verifier/pkg/chartverifier/checks"
	apiReport "github.com/redhat-certification/chart-verifier/pkg/chartverifier/report"
)

var ReportApiVersion = "v1"
var ReportKind = "verify-report"

type InternalReport struct {
	APIReport apiReport.Report
}

type InternalCheckReport struct {
	APICheckReport apiReport.CheckReport
}

func newReport() InternalReport {
	report := InternalReport{}
	report.APIReport = apiReport.Report{Apiversion: ReportApiVersion, Kind: ReportKind}
	report.APIReport.Metadata = apiReport.ReportMetadata{}
	report.APIReport.Metadata.ToolMetadata = apiReport.ToolMetadata{}

	return report
}

func (c *InternalReport) AddCheck(check checks.Check) *InternalCheckReport {
	newCheck := InternalCheckReport{}
	newCheck.APICheckReport = apiReport.CheckReport{}
	newCheck.APICheckReport.Check = apiChecks.CheckName(fmt.Sprintf("%s/%s", check.CheckId.Version, check.CheckId.Name))
	newCheck.APICheckReport.Type = apiChecks.CheckType(check.Type)
	newCheck.APICheckReport.Outcome = apiReport.UnknownOutcomeType
	c.APIReport.Results = append(c.APIReport.Results, &newCheck.APICheckReport)
	return &newCheck
}

func (cr *InternalCheckReport) SetResult(outcome bool, reason string) {
	if outcome {
		cr.APICheckReport.Outcome = apiReport.PassOutcomeType
	} else {
		cr.APICheckReport.Outcome = apiReport.FailOutcomeType
	}
	cr.APICheckReport.Reason = reason
}

func (c *InternalReport) GetApiReport() *apiReport.Report {
	return &c.APIReport
}

func (cr *InternalCheckReport) GetApiCheckReport() *apiReport.CheckReport {
	return &cr.APICheckReport
}
