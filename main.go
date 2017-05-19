package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	nsqLookupdTopics NSQLookupdTopicReport
)

const (
	NSQ_LOOKUPD_HOSTNAME      = "nsq-nsqlookupd"
	NSQ_TOPIC_CONTAINS_FILTER = "_out_"
	NSQ_STATS_FILENAME        = "/go/src/github.com/rlg2161/nsq-topic-cleanup/lastQueueReport.gob"
)

func main() {
	log.SetFlags(log.Lshortfile)
	TopicCleanup(NSQ_STATS_FILENAME)
}

func TopicCleanup(filename string) {

	producersList := []string{}
	topicCounterMap := CreateTopicCounterMap()
	nsqdNodes := CreateNodeProducerList()

	pausedTopicOrChannel := false

	for _, producer := range nsqdNodes.Producers {
		nsqdNodeAddress := producer.BroadcastAddress
		producersList = append(producersList, nsqdNodeAddress)
		for _, topic := range producer.Topics {
			queryString := fmt.Sprintf("http://%s:4151/stats?topic=%s&format=json", nsqdNodeAddress, topic)
			nsqdTopicReport := GetNSQDTopicStats(queryString)
			if nsqdTopicReport.StatusCode == 200 {
				if nsqdTopicReport.Data.Topics[0].Paused {
					pausedTopicOrChannel = true
				}
				for _, channel := range nsqdTopicReport.Data.Topics[0].Channels {
					if channel.Paused {
						pausedTopicOrChannel = true
					}
				}
				if strings.Contains(nsqdTopicReport.Data.Topics[0].TopicName, NSQ_TOPIC_CONTAINS_FILTER) == true {
					topicCounterMap[nsqdTopicReport.Data.Topics[0].TopicName] = nsqdTopicReport.Data.Topics[0].MessageCount
				}
			}
		}
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Println("No existing queue stats -- writing current stats and exiting")
		WriteQueueReport(filename, topicCounterMap)
		os.Exit(1)
	}
	lastQueueStatsMap, err := ReadLastQueueStats(filename)
	if err != nil {
		log.Println(err)
	}

	client := &http.Client{}
	topicsToRemove := []string{}
	if pausedTopicOrChannel != true {
		for topic, _ := range lastQueueStatsMap {
			if lastQueueStatsMap[topic] == topicCounterMap[topic] {
				// topic is no longer being produced since all counts are the same
				err := DeleteTopicFromCluster(client, topic, producersList)
				if err != nil {
					log.Println("Failed to delete topic %s from cluster: %s\n", topic, err.Error())
				} else {
					log.Println("Deleted %s topic from cluster\n", topic)
				}
				topicsToRemove = append(topicsToRemove, topic)
			}
		}
		// do in another loop so we dont screw up iteration over topics list
		for _, topic := range topicsToRemove {
			delete(topicCounterMap, topic)
		}
	} else {
		log.Println("Paused topic or channel found -- not deleting any topics until pause is removed")
	}

	WriteQueueReport(filename, topicCounterMap)
}

func DeleteTopicFromCluster(client *http.Client, topic string, producersList []string) error {
	for _, producer := range producersList {
		err := DeleteNSQDTopic(client, topic, producer)
		if err != nil {
			log.Println("Failed to delete topic from cluster")
			return err
		}
		time.Sleep(5 * time.Second)
	}
	err := DeleteNSQLookupdTopic(client, topic, NSQ_LOOKUPD_HOSTNAME)
	return err
}

func DeleteNSQLookupdTopic(client *http.Client, topic string, address string) error {
	queryString := fmt.Sprintf("http://%s:4161/topic/delete", address)
	urlValues := url.Values{}
	urlValues.Set("topic", topic)
	req, err := http.NewRequest("POST", queryString, nil)
	if err != nil {
		log.Println("Error generating http request: " + err.Error())
		return err
	}
	req.URL.RawQuery = urlValues.Encode()
	_, err = client.Do(req)
	if err != nil {
		log.Println("Error posting request: " + err.Error())
	}
	return err
}

func DeleteNSQDTopic(client *http.Client, topic string, address string) error {
	queryString := fmt.Sprintf("http://%s:4151/topic/delete", address)
	urlValues := url.Values{}
	urlValues.Set("topic", topic)
	req, err := http.NewRequest("POST", queryString, nil)
	if err != nil {
		log.Println("Error generating http request: " + err.Error())
		return err
	}
	req.URL.RawQuery = urlValues.Encode()
	_, err = client.Do(req)
	if err != nil {
		log.Println("Error posting request: " + err.Error())
	}
	return err
}

func GetNSQDTopicStats(queryString string) (nsqdTopicReport NSQDTopicReport) {
	resp, err := http.Get(queryString)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &nsqdTopicReport)
	if err != nil {
		log.Println(err)
	}
	return nsqdTopicReport
}

func CreateTopicCounterMap() map[string]int {
	channelDepthMap := make(map[string]int)

	resp, err := http.Get(fmt.Sprintf("http://%s:4161/topics", NSQ_LOOKUPD_HOSTNAME))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &nsqLookupdTopics)
	if err != nil {
		log.Println(err)
	}

	for _, topic := range nsqLookupdTopics.Topics {
		if strings.Contains(topic, NSQ_TOPIC_CONTAINS_FILTER) {
			channelDepthMap[topic] = 0
		}
	}
	return channelDepthMap
}

func CreateNodeProducerList() (nsqdNodes NSQDNodes) {
	resp, err := http.Get(fmt.Sprintf("http://%s:4161/nodes", NSQ_LOOKUPD_HOSTNAME))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &nsqdNodes)
	if err != nil {
		log.Println(err)
	}
	return nsqdNodes
}

func ReadLastQueueStats(filename string) (map[string]int, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Println("error opening lastQueueReportFile: " + err.Error())
		return nil, err
	}
	gobDecoder := gob.NewDecoder(file)

	lastQueueStatsMap := make(map[string]int)
	if err = gobDecoder.Decode(&lastQueueStatsMap); err != nil {
		log.Println("Error parsing last queue report: " + err.Error())
		return nil, err
	}

	return lastQueueStatsMap, nil

}

func WriteQueueReport(filename string, queueStatsMap map[string]int) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	gobEncoder := gob.NewEncoder(file)
	err = gobEncoder.Encode(queueStatsMap)
	if err != nil {
		return err
	} else {
		return nil
	}
}
