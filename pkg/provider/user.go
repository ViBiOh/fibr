package provider

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (bc User) GetID() string {
	return bc.Login
}

func (bc User) GetSubject() string {
	return bc.Login
}
