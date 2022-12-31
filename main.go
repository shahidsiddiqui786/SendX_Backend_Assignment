package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shahidsiddiqui786/SendX_Backend_Assignment/helper"
)

/* Fetch and Download the url at the filepath ,on error return error */
func DownloadFile(filepath string, url string,retry int) error{

	// Fetching Url Data thorugh http.Get
	response, err := http.Get(url)

	// On Error Trying again or return error
	if err != nil {
		if retry > 0 {
			fmt.Println("Some Error Occured while fetching url trying again!")
			return DownloadFile(filepath, url, retry-1)
		}
		return err
	}

	defer response.Body.Close()
	
	// file creation
	out, err := os.Create(filepath)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, response.Body)
	return err
}

/* Fetch and Download the url with different channels
   if failed adding into retry or failed channel
   if success adding to success
*/
func DownloadConcurrentFile(filename string,request helper.UrlRequest,
	chFailed chan helper.UrlRequest, chRetry chan helper.UrlRequest, 
	chFinished chan bool, chSuccess chan helper.UrlResponse) {

	// Fetching Url Data thorugh http.Get
	response, err := http.Get(request.URI)
	request.RetryLimit = request.RetryLimit - 1

	//since fetched defered a finished task
	defer func() {
        chFinished <- true
    }()

	// On Error adding to respective channel
	if err != nil {
		
		if request.RetryLimit > 0 {
			chRetry <- request
			return
		}

		chFailed <- request
		return
	}
	
	defer response.Body.Close()

	// file creation
	filepath := helper.GeneratePath(filename)
	out, err := os.Create(filepath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		fmt.Println("Wrting error", err)
		return
	}

	//adding to success channel
	successResponse := helper.UrlResponse {
		ID: filename,
		URI: request.URI,
		SourceURI: filepath,
	}
	chSuccess <- successResponse
}

/* fetching concurrently requested urls and separating 
   them into different channels based on success retry failure 
*/
func fetchConcurrentUrls(urls []helper.UrlRequest) ([]helper.UrlRequest,
	 []helper.UrlRequest, []helper.UrlResponse) {

	// four channels created
	// 1. track failure  2.track if need again try 
	// 3. track the success 4. track finsihed fetching or not
    chFailed := make(chan helper.UrlRequest)
	chRetry  := make(chan helper.UrlRequest)
	chSuccess := make(chan helper.UrlResponse)
    chFinished := make(chan bool)

    // Opening all urls concurrently using the 'go' keyword:
    for _, request := range urls {
		uid := uuid.New()
		fmt.Println(request)
		request.RetryLimit = helper.Min(request.RetryLimit, 10)
        go DownloadConcurrentFile(uid.String(), request, chFailed, chRetry, chFinished, chSuccess)
    }

    failedRequest := make([]helper.UrlRequest, 0)
	retryRequest  := make([]helper.UrlRequest, 0)
	successResponse := make([]helper.UrlResponse, 0)

    for i := 0; i < len(urls); {
        select {
        case request := <-chFailed:
            failedRequest = append(failedRequest, request)
		case request := <-chRetry:
            retryRequest = append(retryRequest, request)
		case response := <-chSuccess:
			successResponse = append(successResponse, response)
        case <-chFinished:
            i++
        }
    }

	return retryRequest, failedRequest, successResponse
}

/* Fetching All urls
   Urls already found cache is return directly
*/
func fetchAllUrls(lc *helper.LocalCache,context *gin.Context) {

	var multiRequest helper.UrlMultiRequest
	var multiResponse helper.UrlMultiResponse

	if err := context.BindJSON(&multiRequest); err != nil {
		return
	}

	// test function which removes already found urls in cache
	test := func(request helper.UrlRequest) bool {
		u,er := lc.Read(request.URI)
		if er == nil {
			//also stores the corresponding response of found urls in cache
			multiResponse.URLS = append(multiResponse.URLS,
				helper.UrlResponse{
					URI: u.URL,
					ID: u.ID,
					SourceURI: helper.GeneratePath(u.ID),
				})
			return false
		}
		return true
	}

	requestList := multiRequest.URLS
	//filter will remove urls found already in cache from requestList
	requestList = helper.Filter(requestList,test)
	failedRequest := make([]helper.UrlRequest, 0)

	//trying until all request is finished either failuring or succeeded
	for len(requestList) > 0 {
		retry, failed, success := fetchConcurrentUrls(requestList)
		helper.UpdateCache(lc,success)
		multiResponse.URLS = append(multiResponse.URLS, success...)
		failedRequest = append(failedRequest, failed...)
		requestList = retry
	}

	multiResponse.Failed = append(multiResponse.Failed, failedRequest...)
	context.IndentedJSON(http.StatusCreated, multiResponse)
}


/* Fetching Url and giving back response
   Already Found Urls in cache are return directly
*/
func fetchUrl(lc *helper.LocalCache,context *gin.Context){
	var url helper.UrlRequest

	if err := context.BindJSON(&url); err != nil {
		return
	}

	response := helper.UrlResponse{
		ID: "",
		URI: url.URI,
		SourceURI: "",
	}

	u,er := lc.Read(url.URI)
	if er == nil {
		response.ID = u.ID
		response.SourceURI = helper.GeneratePath(u.ID)

		context.IndentedJSON(http.StatusCreated, response)
		return
	}

	uid := uuid.New()
	filepath := helper.GeneratePath(uid.String())
	err := DownloadFile(filepath, url.URI, helper.Min(url.RetryLimit,10))
	if err != nil {
		context.IndentedJSON(http.StatusNotFound, gin.H{"message": err})
		return
	}

	response.ID = uid.String()
	response.SourceURI = filepath

	testUser := helper.StoreUrl {
		URL: url.URI,
		ID: uid.String(),
	}

	lc.Update(testUser, time.Now().Add(24 * time.Hour).Unix())
	context.IndentedJSON(http.StatusCreated, response)
}


func main() {

	localCache := helper.InitLocalCache(24 * time.Hour)
	router := gin.Default()

	// Create the downloads directory if it doesn't exist
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		os.Mkdir("files", 0755)
	}

	tasks := make(chan helper.Task, 10000)
	for w := 1; w <= 10; w++ {
		go helper.Worker(tasks)
	}

	router.POST("/fetch", func(context *gin.Context) {fetchUrl(localCache,context)})
	router.POST("/fetchAll", func(context *gin.Context) {fetchAllUrls(localCache,context)})

	router.POST("/fetchWorker", func(context *gin.Context) {helper.FetchWorker(localCache,context,tasks)})

	router.Run("localhost:9090")
}