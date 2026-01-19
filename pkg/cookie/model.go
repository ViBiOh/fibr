package cookie

type BasicContent struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (bc BasicContent) GetID() string {
	return bc.Login
}

func (bc BasicContent) GetSubject() string {
	return bc.Login
}
