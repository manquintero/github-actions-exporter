package metrics

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/faubion-hbo/github-actions-exporter/pkg/config"

	"github.com/google/go-github/v45/github"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	runnersOrganizationGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_runner_organization_status",
			Help: "runner status",
		},
		[]string{"organization", "os", "name", "id", "busy"},
	)
)

func getAllOrgRunners(orga string) []*github.Runner {
	var runners []*github.Runner
	opt := &github.ListOptions{PerPage: 200}

	for {
		resp, rr, err := client.Actions.ListOrganizationRunners(context.Background(), orga, opt)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("ListOrganizationRunners ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			if rr.StatusCode == http.StatusForbidden {
				// check Retry-After header if it contains seconds to wait for the next retry
				if retryAfter, e := strconv.ParseInt(rr.Header.Get("Retry-After"), 10, 32); e == nil {
					log.Printf("ListOrganizationRunners Retry-After %d seconds received, going for sleep", retryAfter)
					time.Sleep(time.Duration(retryAfter) * time.Second)
					continue
				}
			}
			log.Printf("ListOrganizationRunners error for org %s: %s", orga, err.Error())
			return runners
		}

		runners = append(runners, resp.Runners...)
		if rr.NextPage == 0 {
			break
		}
		opt.Page = rr.NextPage
	}
	return runners
}

// getRunnersOrganizationFromGithub - return information about runners and their status for an organization
func getRunnersOrganizationFromGithub() {
	for {
		for _, orga := range config.Github.Organizations.Value() {
			runners := getAllOrgRunners(orga)
			for _, runner := range runners {
				if runner.GetStatus() == "online" {
					runnersOrganizationGauge.WithLabelValues(orga, *runner.OS, *runner.Name, strconv.FormatInt(runner.GetID(), 10), strconv.FormatBool(runner.GetBusy())).Set(1)
				} else {
					runnersOrganizationGauge.WithLabelValues(orga, *runner.OS, *runner.Name, strconv.FormatInt(runner.GetID(), 10), strconv.FormatBool(runner.GetBusy())).Set(0)
				}
			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
		runnersOrganizationGauge.Reset()
	}
}
