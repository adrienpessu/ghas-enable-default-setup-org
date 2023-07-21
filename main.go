package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
)

import "github.com/google/go-github/v53/github"

func main() {
	githubToken := flag.String("token", "", "Token should be a personal access githubToken with security_events scope")
	organization := flag.String("organization", "", "GitHub organization")

	flag.Parse()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	opt := &github.RepositoryListByOrgOptions{Type: "all", ListOptions: github.ListOptions{PerPage: 100}}

	// get all pages of results
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, *organization, opt)
		if err != nil {
			return
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	extended := "extended"

	for _, repository := range allRepos {

		languages := getCodeScanningLanguages(*repository.LanguagesURL, *githubToken)

		if languages != nil && len(languages) > 0 {

			options := &github.UpdateDefaultSetupConfigurationOptions{
				State:      "configured",
				Languages:  languages,
				QuerySuite: &extended,
			}

			_, r, err := client.CodeScanning.UpdateDefaultSetupConfiguration(ctx, *organization, *repository.Name, options)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Repository ", repository.GetFullName())
			fmt.Println("Status code: ", r.StatusCode)
			fmt.Println("--------------------------------------------------")
		}
	}
}

func getCodeScanningLanguages(url string, token string) []string {

	languagesCodeScanning := []string{}
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return languagesCodeScanning
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)

	if os.Getenv("DEBUG") == "true" {
		fmt.Println("Secrets scanning alerts: ", res.StatusCode)
	}

	if res.StatusCode == 404 || res.StatusCode == 403 {
		return languagesCodeScanning
	}

	if err != nil {
		fmt.Println(err)
		return languagesCodeScanning
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return languagesCodeScanning
	}

	response := map[string]string{}
	json.Unmarshal([]byte(string(body)), &response)

	for key, _ := range response {
		if key == "Java" || key == "Kotlin" {
			languagesCodeScanning = append(languagesCodeScanning, "java-kotlin")
		} else if key == "JavaScript" || key == "TypeScript" {
			languagesCodeScanning = append(languagesCodeScanning, "javascript-typescript")
		} else if key == "C" || key == "C++" {
			languagesCodeScanning = append(languagesCodeScanning, "c-cpp")
		} else if key == "C#" {
			languagesCodeScanning = append(languagesCodeScanning, "csharp")
		} else if key == "Go" {
			languagesCodeScanning = append(languagesCodeScanning, "go")
		} else if key == "Python" {
			languagesCodeScanning = append(languagesCodeScanning, "python")
		} else if key == "Ruby" {
			languagesCodeScanning = append(languagesCodeScanning, "ruby")
		}
	}

	return removeDuplicateValues(languagesCodeScanning)

}

func removeDuplicateValues(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

type defaultSetupSettings struct {
	Languages  string `json:"languages"`
	State      string `json:"state"`
	QuerySuite string `json:"query_suite"`
}
