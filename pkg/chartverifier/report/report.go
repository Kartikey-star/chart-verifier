package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	FailOutcomeType    OutcomeType = "FAIL"
	PassOutcomeType    OutcomeType = "PASS"
	UnknownOutcomeType OutcomeType = "UNKNOWN"

	JsonReport ReportFormat = "json"
	YamlReport ReportFormat = "yaml"
)

type APIReport interface {
	GetContent(ReportFormat) (string, error)
	SetContent(string) APIReport
	SetURL(url *url.URL) APIReport
	Load() (*Report, error)
}

func NewReport() APIReport {
	r := &Report{}
	r.Init()
	return r
}

func (r *Report) Init() APIReport {
	r.options = &reportOptions{}
	return r
}

func (r *Report) GetContent(format ReportFormat) (string, error) {

	reportContent := ""

	report, loadErr := r.Load()
	if loadErr != nil {
		return "", loadErr
	}

	if format == JsonReport {
		b, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			return "", errors.New(fmt.Sprintf("report json marshal failed : %v", marshalErr))
		}
		reportContent = string(b)
	} else {
		b, marshalErr := yaml.Marshal(report)
		if marshalErr != nil {
			return "", errors.New(fmt.Sprintf("report yaml marshal failed : %v", marshalErr))
		}
		reportContent = string(b)
	}
	return reportContent, nil

}

func (r *Report) SetContent(report string) APIReport {
	r.options.reportString = report
	r.options.reportUrl = nil
	return r
}

func (r *Report) SetURL(url *url.URL) APIReport {
	r.options.reportString = ""
	r.options.reportUrl = url
	return r
}

func (r *Report) Load() (*Report, error) {
	if len(r.options.reportString) > 0 || r.options.reportUrl != nil {
		return r, r.loadReport()
	} else if r.Results == nil {
		return r, errors.New("no report available to load")
	}
	return r, nil
}

func (r *Report) loadReport() error {

	reportString := r.options.reportString
	if len(reportString) == 0 {
		if r.options.reportUrl != nil {
			if r.options.reportUrl.Scheme == "http" || r.options.reportUrl.Scheme == "https" {
				var err error
				reportString, err = loadReportFromRemote(r.options.reportUrl)
				if err != nil {
					return err
				}
			} else {
				return errors.New(fmt.Sprintf("report uri %s: scheme %q not supported", r.options.reportUrl.String(), r.options.reportUrl.Scheme))
			}
		} else {
			return errors.New("no report available to load")
		}
	}

	if strings.HasPrefix(strings.TrimSpace(reportString), "{\"apiversion\":\"v1\"") {
		unMarshalErr := json.Unmarshal([]byte(reportString), r)
		if unMarshalErr != nil {
			return errors.New(fmt.Sprintf("report json ummarshal failed : %v", unMarshalErr))
		}
	} else {
		unMarshalErr := yaml.Unmarshal([]byte(reportString), r)
		if unMarshalErr != nil {
			return errors.New(fmt.Sprintf("report yaml ummarshal failed : %v", unMarshalErr))
		}
	}

	return nil
}

func loadReportFromRemote(url *url.URL) (string, error) {

	if url.Scheme != "http" && url.Scheme != "https" {
		return "", errors.New(fmt.Sprintf("report uri %s: only 'http' and 'https' schemes are supported, but got %s", url, url.Scheme))
	}

	resp, getErr := http.Get(url.String())
	if getErr != nil {
		return "", errors.New(fmt.Sprintf("report uri %s: error reading from url  %v", url, getErr))
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New(fmt.Sprintf("report uri %s: bad response reading from url  %d", url, resp.StatusCode))
	}

	reportBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", errors.New(fmt.Sprintf("report uri %s: error reading reponse body  %v", url, readErr))

	}

	return string(reportBytes), nil
}
