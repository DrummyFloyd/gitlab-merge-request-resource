package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/DrummyFloyd/gitlab-merge-request-resource/pkg"
	"github.com/xanzy/go-gitlab"
)

type Command struct {
	client *gitlab.Client
	runner GitRunner
}

func NewCommand(client *gitlab.Client) *Command {
	return &Command{
		client,
		NewRunner(),
	}
}

func (command *Command) WithRunner(runner GitRunner) *Command {
	command.runner = runner
	return command
}

func (command *Command) Run(destination string, request Request) (Response, error) {
	err := os.MkdirAll(destination, 0755)
	if err != nil {
		return Response{}, err
	}

	mr, _, err := command.client.MergeRequests.GetMergeRequest(request.Source.GetProjectPath(), request.Version.ID, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return Response{}, err
	}

	change, _, err := command.client.MergeRequests.GetMergeRequestChanges(request.Source.GetProjectPath(), request.Version.ID, &gitlab.GetMergeRequestChangesOptions{})
	if err != nil {
		return Response{}, err
	}
	mr.UpdatedAt = request.Version.UpdatedAt

	target, err := command.createRepositoryUrl(mr.TargetProjectID, request.Source.PrivateToken)
	if err != nil {
		return Response{}, err
	}

	commit, _, err := command.client.MergeRequests.GetMergeRequestCommits(mr.ProjectID, mr.IID, &gitlab.GetMergeRequestCommitsOptions{})
	if err != nil {
		return Response{}, err
	}

	err = command.runner.Run("clone", "-c", "http.sslVerify="+strconv.FormatBool(!request.Source.Insecure), "-o", "target", "-b", mr.TargetBranch, target.String(), destination)
	if err != nil {
		return Response{}, err
	}

	os.Chdir(destination)

	err = createDiffPatchFromApi(change)
	if err != nil {
		return Response{}, err
	}

	command.runner.Run("apply", ".git/mr-id_"+strconv.Itoa(mr.ID)+".patch", "--whitespace=nowarn")
	if err != nil {
		return Response{}, err
	}

	notes, _ := json.Marshal(mr)
	err = ioutil.WriteFile(".git/merge-request.json", notes, 0644)
	if err != nil {
		return Response{}, err
	}

	err = ioutil.WriteFile(".git/merge-request-source-branch", []byte(mr.SourceBranch), 0644)
	if err != nil {
		return Response{}, err
	}

	jsonChanges, _ := json.Marshal(change)
	err = ioutil.WriteFile(".git/merge-request-changes", jsonChanges, 0644)
	if err != nil {
		return Response{}, err
	}

	response := Response{Version: request.Version, Metadata: buildMetadata(mr, commit[0])}

	return response, nil
}

func (command *Command) createRepositoryUrl(pid int, token string) (*url.URL, error) {
	project, _, err := command.client.Projects.GetProject(pid, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(project.HTTPURLToRepo)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword("gitlab-ci-token", token)

	return u, nil
}
func createDiffPatchFromApi(mrChange *gitlab.MergeRequest) error {
	fileName := fmt.Sprintf(".git/mr-id_" + strconv.Itoa(mrChange.ID) + ".patch")
	var data string
	_, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	for _, change := range mrChange.Changes {
		if change.NewFile {
			data = fmt.Sprintf(data+"new file mode %s\n--- /dev/null\n+++ b/%s\n%s", change.AMode, change.NewPath, change.Diff)

		} else if change.DeletedFile {
			data = fmt.Sprintf(data+"deleted file mode %s\n--- a/%s\n+++ /dev/null\n%s", change.AMode, change.OldPath, change.Diff)

		} else if change.RenamedFile {
			data = fmt.Sprintf(data+"--- a/%s\n+++ b/%s\n%s", change.OldPath, change.NewPath, change.Diff)

		} else {
			data = fmt.Sprintf(data+"--- a/%s\n+++ b/%s\n%s\n", change.OldPath, change.NewPath, change.Diff)

		}

	}
	ioutil.WriteFile(fileName, []byte(data), 0644)
	return err
}
func buildMetadata(mr *gitlab.MergeRequest, commit *gitlab.Commit) pkg.Metadata {
	return []pkg.MetadataField{
		{
			Name:  "id",
			Value: strconv.Itoa(mr.ID),
		},
		{
			Name:  "iid",
			Value: strconv.Itoa(mr.IID),
		},
		{
			Name:  "sha",
			Value: mr.SHA,
		},
		{
			Name:  "commit title",
			Value: commit.Title,
		},
		{
			Name:  "commit message",
			Value: commit.Message,
		},
		{
			Name:  "title",
			Value: mr.Title,
		},
		{
			Name:  "author",
			Value: mr.Author.Name,
		},
		{
			Name:  "source",
			Value: mr.SourceBranch,
		},
		{
			Name:  "target",
			Value: mr.TargetBranch,
		},
		{
			Name:  "url",
			Value: mr.WebURL,
		},
	}
}
