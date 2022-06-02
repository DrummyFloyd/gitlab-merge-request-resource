	"fmt"
	"log"

	"github.com/DrummyFloyd/gitlab-merge-request-resource/pkg"
	"github.com/xanzy/go-gitlab"
	change, _, err := command.client.MergeRequests.GetMergeRequestChanges(request.Source.GetProjectPath(), request.Version.ID, &gitlab.GetMergeRequestChangesOptions{})
	if err != nil {
		return Response{}, err
	}

	commit, _, err := command.client.MergeRequests.GetMergeRequestCommits(mr.ProjectID, mr.IID, &gitlab.GetMergeRequestCommitsOptions{})
	err = command.runner.Run("clone", "-c", "http.sslVerify="+strconv.FormatBool(!request.Source.Insecure), "-o", "target", "-b", mr.TargetBranch, target.String(), destination)
	os.Chdir(destination)

	err = createDiffPatchFromApi(change)
	command.runner.Run("apply", ".git/mr-id_"+strconv.Itoa(mr.ID)+".patch", "--whitespace=nowarn")
	jsonChanges, _ := json.Marshal(change)
	err = ioutil.WriteFile(".git/merge-request-changes", jsonChanges, 0644)
	if err != nil {
		return Response{}, err
	}

	response := Response{Version: request.Version, Metadata: buildMetadata(mr, commit[0])}
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
			Name:  "commit title",
		{
			Name:  "commit message",
			Value: commit.Message,
		},