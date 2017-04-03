package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

var (
	nsqLookupdTopics NSQLookupdTopicReport
)

func main() {
	// Use cli to get NSQ related env vars
	TopicCleanup("lastQueueReport.gob")

}

func ReadLastQueueStats(filename string) (map[string]int, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		fmt.Println("error opening lastQueueReportFile: " + err.Error())
		return nil, err
	}
	gobDecoder := gob.NewDecoder(file)

	lastQueueStatsMap := make(map[string]int)
	if err = gobDecoder.Decode(&lastQueueStatsMap); err != nil {
		fmt.Println("Error parsing last queue report: " + err.Error())
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

func TopicCleanup(filename string) {

	producersList := []string{}
	topicCounterMap := CreateTopicCounterMap()
	nsqdNodes := CreateNodeProducerList()

	for _, producer := range nsqdNodes.Producers {
		nsqdNodeAddress := producer.BroadcastAddress
		producersList = append(producersList, nsqdNodeAddress)
		for _, topic := range producer.Topics {
			queryString := fmt.Sprintf("http://%s:4151/stats?topic=%s&format=json", nsqdNodeAddress, topic)
			nsqdTopicReport := GetNSQDTopicStats(queryString)
			topicCounterMap[nsqdTopicReport.Data.Topics[0].TopicName] = nsqdTopicReport.Data.Topics[0].MessageCount
		}
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Println("No existing queue stats -- writing current stats and exiting")
		WriteQueueReport(filename, topicCounterMap)
		os.Exit(1)
	}
	lastQueueStatsMap, err := ReadLastQueueStats(filename)
	if err != nil {
		fmt.Println(err)
	}

	topicsToRemove := []string{}
	for topic, _ := range lastQueueStatsMap {
		if lastQueueStatsMap[topic] == topicCounterMap[topic] {
			// topic is no longer being produced since all counts are the same
			err := DeleteTopicFromCluster(topic, producersList)
			if err != nil {
				fmt.Printf("Failed to delete topic % from cluster: %s", topic, err.Error())
			}
			topicsToRemove = append(topicsToRemove, topic)
		}
	}
	// do in another loop so we dont screw up iteration over topics list
	for _, topic := range topicsToRemove {
		delete(topicCounterMap, topic)
	}

	WriteQueueReport(filename, topicCounterMap)
}

func DeleteTopicFromCluster(topic string, producersList []string) error {
	// tombstone the topic w/ nsq-lookupd
	for _, producer := range producersList {
		err := DeleteNSQDTopic(topic, producer)
		if err != nil {
			fmt.Println("Failed to delete topic from cluster")
			return err
		}
	}
	err := DeleteNSQDTopic(topic, "nsq-nsqlookupd")
	return err
}

func DeleteNSQLookupdTopic(topic string, address string) error {
	queryString := fmt.Sprintf("http://%s:4161/delete_topic/", address)
	urlValues := url.Values{}
	urlValues.Set("topic", topic)
	_, err := http.Post(queryString, "application/json", bytes.NewBufferString(urlValues.Encode()))
	return err
}

func DeleteNSQDTopic(topic string, address string) error {
	queryString := fmt.Sprintf("http://%s:4151/topic/delete", address)
	urlValues := url.Values{}
	urlValues.Set("topic", topic)
	_, err := http.Post(queryString, "application/json", bytes.NewBufferString(urlValues.Encode()))
	return err
}

func GetNSQDTopicStats(queryString string) (nsqdTopicReport NSQDTopicReport) {
	resp, err := http.Get(queryString)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(body, &nsqdTopicReport)
	if err != nil {
		fmt.Println(err)
	}
	return nsqdTopicReport
}

func CreateTopicCounterMap() map[string]int {
	channelDepthMap := make(map[string]int)

	resp, err := http.Get("http://nsq-nsqlookupd:4161/topics")
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(body, &nsqLookupdTopics)
	if err != nil {
		fmt.Println(err)
	}

	for _, topic := range nsqLookupdTopics.Topics {
		channelDepthMap[topic] = 0
	}
	return channelDepthMap
}

func CreateNodeProducerList() (nsqdNodes NSQDNodes) {
	resp, err := http.Get("http://nsq-nsqlookupd:4161/nodes")
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(body, &nsqdNodes)
	if err != nil {
		fmt.Println(err)
	}
	return nsqdNodes
}
