package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

type Params struct {
	Endpoint        string // RUNNER_ENDPOINT
	AccessKeyID     string // RUNNER_ACCESSKEYID
	SecretAccessKey string // RUNNER_SECRETACCESSKEY
	Location        string // RUNNER_LOCATION
	Token           string // RUNNER_TOKEN
	Ssl             bool   // RUNNER_SSL
	ScrapperEnabled bool   // RUNNER_SCRAPPERENABLED
}

// NewRunner creates scraper runner
func NewRunner() (*ScraperRunner, error) {
	var params Params
	err := envconfig.Process("runner", &params)
	if err != nil {
		return nil, err
	}

	runner := &ScraperRunner{
		Scraper: scraper.NewMinioScraper(
			params.Endpoint,
			params.AccessKeyID,
			params.SecretAccessKey,
			params.Location,
			params.Token,
			params.Ssl,
		),
		ScrapperEnabled: params.ScrapperEnabled,
	}

	return runner, nil
}

// ScaperRunner prepares data for executor
type ScraperRunner struct {
	ScrapperEnabled bool // RUNNER_SCRAPPERENABLED
	Scraper         scraper.Scraper
}

// Run prepares data for executor
func (r *ScraperRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	// check that the datadir exists
	if execution.ArtifactRequest == nil {
		return result, nil
	}

	_, err = os.Stat(execution.ArtifactRequest.VolumeMountPath)
	if errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	// scrape artifacts first even if there are errors above
	if r.ScrapperEnabled && len(execution.ArtifactRequest.Dirs) != 0 {
		directories := execution.ArtifactRequest.Dirs
		for i := range directories {
			directories[i] = filepath.Join(execution.ArtifactRequest.VolumeMountPath, directories[i])
		}
		err := r.Scraper.Scrape(execution.Id, directories)
		if err != nil {
			return result.WithErrors(fmt.Errorf("scrape artifacts error: %w", err)), nil
		}
	}

	return result.WithErrors(err), nil
}
