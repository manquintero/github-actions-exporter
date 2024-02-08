package metrics

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faubion-hbo/github-actions-exporter/pkg/config"

	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	workflowBillGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_workflow_usage_seconds",
			Help: "Number of billable seconds used by a specific workflow during the current billing cycle. Any job re-runs are also included in the usage. Only apply to workflows in private repositories that use GitHub-hosted runners.",
		},
		[]string{"repo", "id", "node_id", "name", "state", "os"},
	)
)

// getBillableFromGithub - return billable informations for MACOS, WINDOWS and UBUNTU runners.
func getBillableFromGithub() {
	for {
		for _, repo := range repositories {
			for k, v := range workflows[repo] {
				r := strings.Split(repo, "/")

				for {
					usage, resp, err := client.Actions.GetWorkflowUsageByID(context.Background(), r[0], r[1], k)
					if rl_err, ok := err.(*github.RateLimitError); ok {
						log.Printf("GetWorkflowUsageByID ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
						time.Sleep(time.Until(rl_err.Rate.Reset.Time))
						continue
					} else if err != nil {
						if resp.StatusCode == http.StatusForbidden {
							if retryAfterSeconds, e := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 32); e == nil {
								delaySeconds := retryAfterSeconds + (60 * rand.Int63n(randomDelaySeconds))
								log.Printf("GetWorkflowUsageByID Retry-After %d seconds received, sleeping for %d", retryAfterSeconds, delaySeconds)
								time.Sleep(time.Duration(delaySeconds) * time.Second)
								continue
							}
						}
						log.Printf("GetWorkflowUsageByID error for %s: %s", repo, err)
						break
					}
					workflowBillGauge.WithLabelValues(repo, strconv.FormatInt(*v.ID, 10), *v.NodeID, *v.Name, *v.State, "MACOS").Set(float64(usage.GetBillable().MacOS.GetTotalMS()) / 1000)
					workflowBillGauge.WithLabelValues(repo, strconv.FormatInt(*v.ID, 10), *v.NodeID, *v.Name, *v.State, "WINDOWS").Set(float64(usage.GetBillable().Windows.GetTotalMS()) / 1000)
					workflowBillGauge.WithLabelValues(repo, strconv.FormatInt(*v.ID, 10), *v.NodeID, *v.Name, *v.State, "UBUNTU").Set(float64(usage.GetBillable().Ubuntu.GetTotalMS()) / 1000)
					break
				}

			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * 5 * time.Second)
	}
}
