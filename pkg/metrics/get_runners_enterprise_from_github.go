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
	runnersEnterpriseGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_runner_enterprise_status",
			Help: "runner status",
		},
		[]string{"os", "name", "id"},
	)
)

func getAllEnterpriseRunners() []*github.Runner {
	var runners []*github.Runner
	opt := &github.ListOptions{PerPage: 200}

	for {
		resp, rr, err := client.Enterprise.ListRunners(context.Background(), config.EnterpriseName, nil)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("ListRunners ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			if rr.StatusCode == http.StatusForbidden {
				log.Printf("DocumentationURL: %s", err.(*github.ErrorResponse).DocumentationURL)
				if retryAfterSeconds, e := strconv.ParseInt(rr.Header.Get("Retry-After"), 10, 32); e == nil {
					log.Printf("ListRunners Retry-After %d seconds received, going for sleep", retryAfterSeconds)
					time.Sleep(time.Duration(retryAfterSeconds) * time.Second)
					continue
				}
			}
			log.Printf("ListRunners error for enterprise %s: %s", config.EnterpriseName, err.Error())
			return nil
		}

		runners = append(runners, resp.Runners...)
		if rr.NextPage == 0 {
			break
		}
		opt.Page = rr.NextPage
	}

	return runners
}

func getRunnersEnterpriseFromGithub() {
	if config.EnterpriseName == "" {
		return
	}
	for {
		runners := getAllEnterpriseRunners()

		for _, runner := range runners {
			var integerStatus float64
			if integerStatus = 0; runner.GetStatus() == "online" {
				integerStatus = 1
			}
			runnersEnterpriseGauge.WithLabelValues(*runner.OS, *runner.Name, strconv.FormatInt(runner.GetID(), 10)).Set(integerStatus)
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}
