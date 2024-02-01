package metrics

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"

	"github.com/faubion-hbo/github-actions-exporter/pkg/config"
)

var (
	repositories  []string
	repos_per_org map[string][]string
	workflows     map[string]map[int64]github.Workflow
)

func countAllReposForOrg(orga string) int {
	for {
		organization, _, err := client.Organizations.Get(context.Background(), orga)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("Organizations ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			log.Printf("Get error for %s: %s", orga, err.Error())
			break
		}
		return *organization.PublicRepos + *organization.TotalPrivateRepos + *organization.OwnedPrivateRepos
	}
	return -1
}

func getAllReposForOrg(orga string) []string {
	var all_repos []string

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
			Page:    0,
		},
	}
	for {
		repos_page, resp, err := client.Repositories.ListByOrg(context.Background(), orga, opt)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("ListByOrg ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			log.Printf("ListByOrg error for %s: %s", orga, err.Error())
			break
		}
		for _, repo := range repos_page {
			if *repo.Disabled || *repo.Archived {
				log.Printf("Skipping Archived or Disabled repo %s", *repo.FullName)
				continue
			}
			all_repos = append(all_repos, *repo.FullName)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	return all_repos
}

func getAllWorkflowsForRepo(owner string, repo string) map[int64]github.Workflow {
	res := make(map[int64]github.Workflow)

	opt := &github.ListOptions{
		PerPage: 100,
		Page:    0,
	}

	for {
		workflows_page, resp, err := client.Actions.ListWorkflows(context.Background(), owner, repo, opt)
		if rl_err, ok := err.(*github.RateLimitError); ok {
			log.Printf("ListWorkflows ratelimited. Pausing until %s", rl_err.Rate.Reset.Time.String())
			time.Sleep(time.Until(rl_err.Rate.Reset.Time))
			continue
		} else if err != nil {
			log.Printf("ListWorkflows error for %s: %s", repo, err.Error())
			return res
		}
		for _, w := range workflows_page.Workflows {
			res[*w.ID] = *w
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return res
}

func periodicGithubFetcher() {
	for {

		// Fetch repositories (if dynamic)
		var repos_to_fetch []string
		var current_repos_per_org = make(map[string][]string)
		var r []string

		if len(config.Github.Repositories.Value()) > 0 {
			repos_to_fetch = config.Github.Repositories.Value()
		} else {
			for _, orga := range config.Github.Organizations.Value() {
				repos, exist := repos_per_org[orga]
				previousCount := len(repos)
				currentCount := countAllReposForOrg(orga)
				if !exist || previousCount != currentCount {
					log.Printf("Repo count updated from %d to %d", previousCount, currentCount)
					r = getAllReposForOrg(orga)
				} else {
					log.Printf("Skipping getAllReposForOrg, repo count unchanged: %d", previousCount)
					r = repos_per_org[orga]
				}
				repos_to_fetch = append(repos_to_fetch, r...)
				current_repos_per_org[orga] = r
			}
		}
		repositories = repos_to_fetch
		repos_per_org = current_repos_per_org

		// Fetch workflows
		non_empty_repos := make([]string, 0)
		ww := make(map[string]map[int64]github.Workflow)
		for _, repo := range repos_to_fetch {
			r := strings.Split(repo, "/")
			workflows_for_repo := getAllWorkflowsForRepo(r[0], r[1])
			if len(workflows_for_repo) == 0 {
				continue
			}
			non_empty_repos = append(non_empty_repos, repo)
			ww[repo] = workflows_for_repo
			log.Printf("Fetched %d workflows for repository %s", len(ww[repo]), repo)
		}
		repositories = non_empty_repos
		workflows = ww

		time.Sleep(time.Duration(config.Github.Refresh) * 5 * time.Second)
	}
}
