package entity

type Shortlink struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Short  string `json:"short"`
	Long   string `json:"long"`
}
