package health

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Check() map[string]string {
	return map[string]string{"status": "ok"}
}
