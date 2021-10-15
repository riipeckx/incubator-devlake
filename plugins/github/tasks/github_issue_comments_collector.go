package tasks

import (
	"fmt"
	"net/http"

	"github.com/merico-dev/lake/logger"
	lakeModels "github.com/merico-dev/lake/models"
	"github.com/merico-dev/lake/plugins/core"
	"github.com/merico-dev/lake/plugins/github/models"
	"github.com/merico-dev/lake/utils"
	"gorm.io/gorm/clause"
)

type ApiIssueCommentResponse []IssueComment

type IssueComment struct {
	GithubId int `json:"id"`
	Body     string
	User     struct {
		Login string
	}
	GithubCreatedAt string `json:"created_at"`
}

func CollectIssueComments(owner string, repositoryName string, issue *models.GithubIssue, scheduler *utils.WorkerScheduler) error {
	githubApiClient := CreateApiClient()
	getUrl := fmt.Sprintf("repos/%v/%v/issues/%v/comments", owner, repositoryName, issue.Number)
	return githubApiClient.FetchWithPaginationAnts(getUrl, 100, 1, scheduler,
		func(res *http.Response) error {
			githubApiResponse := &ApiIssueCommentResponse{}
			if res.StatusCode == 200 {
				err := core.UnmarshalResponse(res, githubApiResponse)
				if err != nil {
					logger.Error("Error: ", err)
					return err
				}
				for _, comment := range *githubApiResponse {
					githubComment := &models.GithubIssueComment{
						GithubId:        comment.GithubId,
						IssueId:         issue.GithubId,
						Body:            comment.Body,
						AuthorUsername:  comment.User.Login,
						GithubCreatedAt: utils.ConvertStringToSqlNullTime(comment.GithubCreatedAt),
					}
					err = lakeModels.Db.Clauses(clause.OnConflict{
						UpdateAll: true,
					}).Create(&githubComment).Error
					if err != nil {
						logger.Error("Could not upsert: ", err)
					}
				}
			} else {
				fmt.Println("INFO: PR Comment collection >>> res.Status: ", res.Status)
			}
			return nil
		})
}
