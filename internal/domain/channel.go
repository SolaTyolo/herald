package domain

type ChannelType string

const (
	ChannelInApp ChannelType = "in_app"
	ChannelEmail ChannelType = "email"
	ChannelSMS   ChannelType = "sms"
	ChannelPush  ChannelType = "push"
	ChannelChat  ChannelType = "chat"
)

type StepType string

const (
	StepInApp    StepType = "in_app"
	StepEmail    StepType = "email"
	StepSMS      StepType = "sms"
	StepPush     StepType = "push"
	StepChat     StepType = "chat"
	StepDelay    StepType = "delay"
	StepDigest   StepType = "digest"
	StepThrottle StepType = "throttle"
)

func (s StepType) IsChannel() bool {
	switch s {
	case StepInApp, StepEmail, StepSMS, StepPush, StepChat:
		return true
	default:
		return false
	}
}

func (s StepType) Channel() ChannelType {
	switch s {
	case StepInApp:
		return ChannelInApp
	case StepEmail:
		return ChannelEmail
	case StepSMS:
		return ChannelSMS
	case StepPush:
		return ChannelPush
	case StepChat:
		return ChannelChat
	default:
		return ""
	}
}

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobDelayed   JobStatus = "delayed"
	JobSkipped   JobStatus = "skipped"
	JobMerged    JobStatus = "merged"
)

type NotificationStatus string

const (
	NotificationPending   NotificationStatus = "pending"
	NotificationRunning   NotificationStatus = "running"
	NotificationCompleted NotificationStatus = "completed"
	NotificationFailed    NotificationStatus = "failed"
)
