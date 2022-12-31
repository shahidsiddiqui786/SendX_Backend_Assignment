package helper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)


func Worker(tasks <-chan Task) {
	for task := range tasks {
		downloadFile(task)
	}
}


func FetchWorker(lc *LocalCache,context *gin.Context, tasks chan<- Task){
	var url UrlRequest

	if err := context.BindJSON(&url); err != nil {
		return
	}

	response := UrlResponse{
		ID: "",
		URI: url.URI,
		SourceURI: "",
	}

	u,er := lc.Read(url.URI)
	if er == nil {
		response.ID = u.ID
		response.SourceURI = GeneratePath(u.ID)

		context.IndentedJSON(http.StatusCreated, response)
		return
	}

	uid := uuid.New()
	filepath := GeneratePath(uid.String())
	task := Task{Request: url,ID: uid.String()}
	tasks <- task

	for i:=1 ;  ; i++ {
		_, err := os.Stat(filepath); 
		if(!os.IsNotExist(err)) {
			response.ID = uid.String()
			response.SourceURI = filepath

			testUser := StoreUrl {
				URL: url.URI,
				ID: uid.String(),
			}

			lc.Update(testUser, time.Now().Add(24 * time.Hour).Unix())
			context.IndentedJSON(http.StatusCreated, response)
			break;
		}
	}
}


func downloadFile(task Task) {
	
	resp, err := http.Get(task.Request.URI)
	task.Request.RetryLimit = task.Request.RetryLimit - 1

	// On Error Trying again or return error
	if err != nil {
		if task.Request.RetryLimit > 0 {
			fmt.Println("Some Error Occured while fetching url trying again!")
			downloadFile(task)
		}
		return
	}
	defer resp.Body.Close()

    // file creation
	out, err := os.Create(GeneratePath(task.ID))
	if err != nil {
		return
	}
	defer out.Close()

	// Writing the body to file
	io.Copy(out, resp.Body)
}