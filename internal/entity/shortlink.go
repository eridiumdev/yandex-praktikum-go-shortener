package entity

type Shortlink struct {
	UID     string `json:"id"`
	UserUID string `json:"user_id"`
	Short   string `json:"short"`
	Long    string `json:"long"`

	CorrelationID string `json:"correlation_id"`
}
