package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faubion-hbo/github-actions-exporter/pkg/config"

	"github.com/google/go-github/v45/github"
)

// getFieldValue return value from run element which corresponds to field
func getFieldValue(repo string, run github.WorkflowRun, field string) string {
	switch field {
	case "repo":
		return repo
	case "id":
		runId := run.ID
		if runId == nil {
			return "0"
		}
		return strconv.FormatInt(*runId, 10)
	case "node_id":
		nodeId := run.NodeID
		if nodeId == nil {
			return "<empty>"
		}
		return *nodeId
	case "head_branch":
		headBranch := run.HeadBranch
		if headBranch == nil {
			return "<empty>"
		}
		return *headBranch
	case "head_sha":
		headSha := run.HeadSHA
		if headSha == nil {
			return "<empty>"
		}
		return *headSha
	case "run_number":
		runNumber := run.RunNumber
		if runNumber == nil {
			return "0"
		}
		return strconv.Itoa(*runNumber)
	case "workflow_id":
		workflowId := run.WorkflowID
		if workflowId == nil {
			return "0"
		}
		return strconv.FormatInt(*workflowId, 10)
	case "workflow":
		r, exist := workflows[repo]
		if !exist {
			log.Printf("Couldn't fetch repo '%s' from workflow cache.", repo)
			return "unknown"
		}
		workflowId := run.WorkflowID
		if workflowId == nil {
			log.Printf("Couldn't fetch workflow for repo '%s' from workflow cache because WorkflowID was missing from the passed in run object.", repo)
			return "unknown"
		}
		w, exist := r[*workflowId]
		if !exist {
			log.Printf("Couldn't fetch repo '%s', workflow '%d' from workflow cache.", repo, *workflowId)
			return "unknown"
		}
		return *w.Name
	case "event":
		runEvent := run.Event
		if runEvent == nil {
			return "<empty>"
		}
		return *runEvent
	case "status":
		runStatus := run.Status
		if runStatus == nil {
			return "<empty>"
		}
		return *runStatus
	}
	log.Printf("Tried to fetch invalid field '%s'", field)
	return ""
}

var debug = false

func getRelevantFields(repo string, run *github.WorkflowRun) []string {
	relevantFields := strings.Split(config.WorkflowFields, ",")
	if debug {
		log.Print("relevantFields=", relevantFields)
	}
	result := make([]string, len(relevantFields))
	for i, field := range relevantFields {
		result[i] = getFieldValue(repo, *run, field)
		if debug {
			var err error
			var runBytes []byte
			if runBytes, err = json.Marshal(*run); err != nil {
				log.Fatalln("failed to json.Marshal() the github.WorkflowRun type into string:", err)
			}
			bytesCompact := &bytes.Buffer{}
			if err = json.Compact(bytesCompact, runBytes); err != nil {
				log.Fatalln("failed to json.Compact() the github.WorkflowRun string:", err)
			}
			log.Print("repo=", repo, ";", "field=", field, ";", "runJson=", bytesCompact.String())
		}
	}
	if debug {
		log.Print("result=", result)
	}
	return result
}

func getRecentWorkflowRuns(owner string, repo string) []*github.WorkflowRun {
	// FIXME: make the window dynamic
	window_start := time.Now().Add(time.Duration(-8) * time.Hour).Format(time.RFC3339)
	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Created:     ">=" + window_start,
	}

	var runs []*github.WorkflowRun
	for {
		workflow_runs, response, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), owner, repo, opt)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("ListRepositoryWorkflowRuns ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			if response.StatusCode == http.StatusForbidden {
				log.Printf("DocumentationURL: %s", err.(*github.ErrorResponse).DocumentationURL)
				if retryAfterSeconds, e := strconv.ParseInt(response.Header.Get("Retry-After"), 10, 32); e == nil {
					log.Printf("ListRepositoryWorkflowRuns Retry-After %d seconds received, going for sleep", retryAfterSeconds)
					time.Sleep(time.Duration(retryAfterSeconds) * time.Second)
					continue
				}
			}
			log.Printf("ListRepositoryWorkflowRuns error for repo %s/%s: %s [%d]", owner, repo, err, response.StatusCode)
			return runs
		}

		runs = append(runs, workflow_runs.WorkflowRuns...)
		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
	}

	return runs
}

func getRunUsage(owner string, repo string, runId int64) *github.WorkflowRunUsage {
	for {
		resp, _, err := client.Actions.GetWorkflowRunUsageByID(context.Background(), owner, repo, runId)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("GetWorkflowRunUsageByID ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			log.Printf("GetWorkflowRunUsageByID error for repo %s/%s and runId %d: %s", owner, repo, runId, err.Error())
			return nil
		}
		return resp
	}
}

// getWorkflowRunsFromGithub - return informations and status about a workflow
func getWorkflowRunsFromGithub() {
	for {
		for _, repo := range repositories {
			r := strings.Split(repo, "/")
			runs := getRecentWorkflowRuns(r[0], r[1])

			for _, run := range runs {
				var s float64 = 0
				if run.GetConclusion() == "success" {
					s = 1
				} else if run.GetConclusion() == "skipped" {
					s = 2
				} else if run.GetConclusion() == "in_progress" {
					s = 3
				} else if run.GetConclusion() == "queued" {
					s = 4
				}

				fields := getRelevantFields(repo, run)

				workflowRunStatusGauge.WithLabelValues(fields...).Set(s)

				var run_usage *github.WorkflowRunUsage = nil
				if config.Metrics.FetchWorkflowRunUsage {
					run_usage = getRunUsage(r[0], r[1], *run.ID)
				}
				if run_usage == nil { // Fallback for Github Enterprise
					created := run.CreatedAt.Time.Unix()
					updated := run.UpdatedAt.Time.Unix()
					elapsed := updated - created
					workflowRunDurationGauge.WithLabelValues(fields...).Set(float64(elapsed * 1000))
				} else {
					workflowRunDurationGauge.WithLabelValues(fields...).Set(float64(run_usage.GetRunDurationMS()))
				}
			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}
