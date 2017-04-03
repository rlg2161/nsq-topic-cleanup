package main

type NSQDTopicReport struct {
	StatusCode int    `json:"status_code"`
	StatusTxt  string `json:"status_txt"`
	Data       Data   `json:"data"`
}

type Data struct {
	Version   string  `json:"version"`
	Health    string  `json:"health"`
	StartTime int     `json:"start_time"`
	Topics    []Topic `json:"topics"`
}

type Topic struct {
	TopicName            string               `json:"topic_name"`
	Channels             []Channel            `json:"channels"`
	Depth                int                  `json:"depth"`
	BackendDepth         int                  `json:"backend_depth"`
	MessageCount         int                  `json:"message_count"`
	Paused               bool                 `json:"paused"`
	E2EProcessingLatency E2EProcessingLatency `json:"e2e_processing_latency"`
}

type Channel struct {
	ChannelName          string               `json:"channel_name"`
	Depth                int                  `json:"depth"`
	BackendDepth         int                  `json:"backend_depth"`
	InFlightCount        int                  `json:"in_flight_count"`
	DeferredCount        int                  `json:"deferred_count"`
	MessageCount         int                  `json:"message_count"`
	RequeueCount         int                  `json:"requeue_count"`
	TimeoutCount         int                  `json:"timeout_count"`
	Clients              []Client             `json:"clients"`
	Paused               bool                 `json:"paused"`
	E2EProcessingLatency E2EProcessingLatency `json:"e2e_processing_latency"`
}

type Client struct {
	Name                          string `json:"name"`
	ClientID                      string `json:"client_id"`
	Hostname                      string `json:"hostname"`
	Version                       string `json:"version"`
	RemoteAddress                 string `json:"remote_address"`
	State                         int    `json:"state"`
	ReadyCount                    int    `json:"ready_count"`
	InFlightCount                 int    `json:"in_flight_count"`
	MessageCount                  int    `json:"message_count"`
	FinishCount                   int    `json:"finish_count"`
	RequeueCount                  int    `json:"requeue_count"`
	ConnectTs                     int    `json:"connect_ts"`
	SampleRate                    int    `json:"sample_rate"`
	Deflate                       bool   `json:"deflate"`
	Snappy                        bool   `json:"snappy"`
	UserAgent                     string `json:"user_agent"`
	TLS                           bool   `json:"tls"`
	TLSCipherSuite                string `json:"tls_cipher_suite"`
	TLSVersion                    string `json:"tls_version"`
	TLSNegotiatedProtocol         string `json:"tls_negotiated_protocol"`
	TLSNegotiatedProtocolIsMutual bool   `json:"tls_negotiated_protocol_is_mutual"`
}
type E2EProcessingLatency struct {
	Count       int         `json:"count"`
	Percentiles interface{} `json:"percentiles"`
}
