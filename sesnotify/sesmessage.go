package main

type SesMessage struct {
	NotificationType string                 `json:"notificationType"`
	Content          string                 `json:"content"`
	Mail             Mail                   `json:"mail"`
	Bounce           Bounce                 `json:"bounce"`
	Complaint        Complaint              `json:"complaint"`
	Receipt          map[string]interface{} `json:"receipt"`
}

type Bounce struct {
	BounceType        string          `json:"bounceType"`
	BounceSubType     string          `json:"bounceSubType"`
	BouncedRecipients []MailRecipient `json:"bouncedRecipients"`
	Timestamp         string          `json:"timestamp"`
	FeedbackID        string          `json:"feedbackId"`
	RemoteMtaIp       string          `json:"remoteMtaIp"`
}

type MailRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

type Complaint struct {
	UserAgent             string          `json:"userAgent"`
	ComplainedRecipients  []MailRecipient `json:"complainedRecipients"`
	ComplaintFeedbackType string          `json:"complaintFeedbackType"`
	ArrivalDate           string          `json:"arrivalDate"`
	Timestamp             string          `json:"timestamp"`
	FeedbackID            string          `json:"feedbackId"`
}

type Mail struct {
	Timestamp        string                 `json:"timestamp"`
	Source           string                 `json:"source"`
	MessageId        string                 `json:"messageId"`
	HeadersTruncated bool                   `json:"headersTruncated"`
	Destination      []string               `json:"destination"`
	Headers          []map[string]string    `json:"headers"`
	CommonHeaders    map[string]interface{} `json:"commonHeaders"`
}
