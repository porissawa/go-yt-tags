package main

// SO, I'm getting similar times in this compared to the JS one.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Metadata structures the required fields to display to users
type Metadata struct {
	ChannelName string
	VideoTitle  string
	Keywords    []string
}

func main() {
	start := time.Now()
	var wg sync.WaitGroup
	metadataChannel := make(chan string)

	fetchChannelVideoTags("https://www.youtube.com/channel/UC0fGGprihDIlQ3ykWvcb9hg/videos", &wg, metadataChannel)
	wg.Wait()
	var data []string
	for res := range metadataChannel {
		data = append(data, res)
	}
	duration := time.Since(start)
	fmt.Println(data)
	fmt.Println("Exection took", duration)
}

func fetchVideoTags(videoID string, channel chan string, wg *sync.WaitGroup, shouldCloseChan bool) {
	videoData := fetchVideo(videoID, wg)
	parsedMetadata := parseVideoMetadata(videoData)
	data := marshalMetadata(parsedMetadata)
	channel <- string(data)
	if shouldCloseChan {
		close(channel)
	}
}

func fetchVideo(videoID string, wg *sync.WaitGroup) []byte {
	defer wg.Done()
	resp, err := http.Get("https://youtube.com/watch?v=" + videoID)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	return body
}

func parseVideoMetadata(data []byte) Metadata {
	cre := regexp.MustCompile(`"ownerChannelName":\"(.*?)\"`)
	channelNameMatch := string(cre.Find(data))
	channelName := strings.Replace(channelNameMatch[19:], "\"", "", -1)

	tre := regexp.MustCompile(`\<title\>(.*?)\<\/title\>`)
	var videoTitleMatch string
	// sometimes a title fails to be found.
	// No idea why since looping over the same data finds it on a
	// different iteration
	for len(videoTitleMatch) < 1 {
		videoTitleMatch = string(tre.Find(data))
	}
	videoTitle := strings.Replace(videoTitleMatch[7:len(videoTitleMatch)-18], "\"", "", -1)

	// a video may not have any keywords
	var keywords []string
	kre := regexp.MustCompile(`keywords\":\[(.*?)\]`)
	if len(kre.Find(data)) > 0 {
		err := json.Unmarshal(kre.Find(data)[10:], &keywords)
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}

	return Metadata{
		ChannelName: channelName,
		VideoTitle:  videoTitle,
		Keywords:    keywords,
	}
}

func marshalMetadata(metadata Metadata) []byte {
	m, err := json.Marshal(metadata)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	return m
}

func fetchChannelVideoTags(channelURL string, wg *sync.WaitGroup, channel chan string) {
	resp, err := http.Get(channelURL)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	videoIDList := parseVideoIDList(body)
	// var list []string
	for i, v := range videoIDList {
		wg.Add(1)
		isLast := i == len(videoIDList)-1
		go fetchVideoTags(v, channel, wg, isLast)
	}
}

func parseVideoIDList(data []byte) []string {
	re := regexp.MustCompile(`videoIds\":\[(.*?)\]`)
	videoIDsMatch := re.FindAll(data, -1)
	var videoIDs []string
	for _, v := range videoIDsMatch {
		vd := string(v)
		parsed := vd[12 : len(vd)-2]
		videoIDs = append(videoIDs, parsed)
	}
	return unique(videoIDs)
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
